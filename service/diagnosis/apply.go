package diagnosis

import (
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Apply 校验用户授权并异步执行选中的修复动作。
func (s *Service) Apply(actor service.Actor, id string, input ApplyInput) service.Result {
	session := s.store.get(id)
	if session == nil {
		return service.Fail(constant.CodeDiagnosisNotFound)
	}
	if session.Actor.Username != actor.Username && !actor.IsAdmin() {
		return service.Fail(constant.CodeDiagnosisNoPermission)
	}
	if session.expired() {
		return service.Fail(constant.CodeDiagnosisExpired)
	}
	if session.Status != StatusSucceeded {
		return service.Fail(constant.CodeDiagnosisNotReady)
	}
	if !session.canApply() {
		if apply := session.getApply(); apply != nil && apply.Status == StatusRunning {
			return service.Fail(constant.CodeDiagnosisApplyBusy)
		}
		return service.Fail(constant.CodeDiagnosisNotReady)
	}
	if len(input.ActionIDs) == 0 {
		return service.Fail(constant.CodeDiagnosisBadRequest)
	}

	selected := make([]*Action, 0, len(input.ActionIDs))
	seen := make(map[string]struct{}, len(input.ActionIDs))
	for _, actionID := range input.ActionIDs {
		if _, ok := seen[actionID]; ok {
			continue
		}
		seen[actionID] = struct{}{}
		action := session.getAction(actionID)
		if action == nil {
			return service.Fail(constant.CodeDiagnosisActionInvalid)
		}
		spec, ok := s.actionSpec(actionID)
		if !ok {
			return service.Fail(constant.CodeDiagnosisActionInvalid)
		}
		if spec.AdminOnly && !actor.IsAdmin() {
			return service.Fail(constant.CodeDiagnosisNoPermission)
		}
		params := copyParams(action.Params)
		if override, ok := input.ParamOverrides[actionID]; ok {
			for key, value := range override {
				params[key] = value
			}
		}
		action.Params = params

		scope, err := s.actionScopeForAction(&Scope{
			Actor:    session.Actor,
			Platform: session.Scope == ScopePlatform,
			RoomID:   session.RoomID,
		}, actionID, params)
		if err != nil {
			logger.Logger.Warnf("诊断动作上下文无效, action: %s, err: %v", actionID, err)
			return service.Fail(constant.CodeDiagnosisActionPrecondition)
		}
		if err = spec.Validate(context.Background(), s, scope, params); err != nil {
			logger.Logger.Warnf("诊断动作前置条件不满足, action: %s, err: %v", actionID, err)
			return service.Fail(constant.CodeDiagnosisActionPrecondition)
		}
		if err = s.requireEvidence(actionID, session, action.EvidenceIDs, params); err != nil {
			return service.Fail(constant.CodeDiagnosisActionPrecondition)
		}
		if err = checkConfirmation(spec, input, scope); err != nil {
			return service.Fail(constant.CodeDiagnosisConfirmRequired)
		}
		selected = append(selected, action)
	}

	sort.SliceStable(selected, func(i, j int) bool {
		si, _ := s.actionSpec(selected[i].ActionID)
		sj, _ := s.actionSpec(selected[j].ActionID)
		return si.Order < sj.Order
	})

	lockID := session.RoomID
	if session.Scope == ScopePlatform {
		lockID = -1
	}
	lock := s.roomLock(lockID)
	if !lock.TryLock() {
		return service.Fail(constant.CodeDiagnosisApplyBusy)
	}
	locked := true
	defer func() {
		if locked {
			lock.Unlock()
		}
	}()

	random, err := utils.GenerateRandomKey(12)
	if err != nil {
		return service.Fail(constant.CodeDiagnosisDBError)
	}
	applyID := "apply_" + random
	now := time.Now()
	session.setApply(&ApplyState{
		ApplyID:   applyID,
		Status:    StatusRunning,
		StartedAt: now,
		Results:   []ActionResult{},
	})
	for _, action := range selected {
		action.Status = ActionStatusRunning
		session.setActionStatus(action.ActionID, ActionStatusRunning)
	}
	session.addApplyEvent(EventApplyStart, map[string]any{
		"applyID":   applyID,
		"actionIDs": actionIDs(selected),
		"startedAt": now,
	})

	// 落库动作审计记录，执行过程中逐条更新。
	records := make(map[string]*models.DiagnosisActionRecord, len(selected))
	if s.diagnosisDao != nil {
		for _, action := range selected {
			params, _ := json.Marshal(action.Params)
			record := &models.DiagnosisActionRecord{
				DiagnosisID: session.ID,
				ActionID:    action.ActionID,
				Params:      string(params),
				Risk:        action.Risk,
				Status:      ActionStatusRunning,
				Operator:    actor.Username,
				CreatedAt:   now,
			}
			if err = s.diagnosisDao.CreateActionRecord(record); err != nil {
				logger.Logger.Warnf("创建动作审计记录失败: %v", err)
			} else {
				records[action.ActionID] = record
			}
		}
	}

	go func() {
		defer lock.Unlock()
		s.runApply(session, selected, records)
	}()
	locked = false

	return service.Success(ApplyData{
		ApplyID:   applyID,
		Status:    StatusRunning,
		EventsURL: fmt.Sprintf("/%s/diagnosis/%s/apply/events", constant.ApiVersion, session.ID),
	})
}

func (s *Service) runApply(session *Session, actions []*Action, records map[string]*models.DiagnosisActionRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	overall := StatusSucceeded
	stopped := false
	for index, action := range actions {
		if stopped {
			result := ActionResult{ActionID: action.ActionID, Status: ActionStatusSkipped, Message: "前置动作失败，已跳过"}
			session.appendApplyResult(result)
			session.setActionResult(result)
			s.updateActionRecord(records[action.ActionID], result)
			session.addApplyEvent(EventActionDone, result)
			continue
		}

		session.addApplyEvent(EventActionStart, map[string]any{
			"actionID":  action.ActionID,
			"index":     index,
			"startedAt": time.Now(),
		})
		result := s.applyAction(ctx, session, action)
		session.appendApplyResult(result)
		session.setActionResult(result)
		s.updateActionRecord(records[action.ActionID], result)
		session.addApplyEvent(EventActionDone, result)

		if result.Status != ActionStatusSucceeded {
			overall = StatusFailed
			stopped = true
			continue
		}
		if !result.Verified {
			overall = "partial"
		}
	}

	session.finishApply(overall)
	session.addApplyEvent(EventApplyEnd, map[string]any{
		"status":     overall,
		"results":    session.getApply().Results,
		"finishedAt": time.Now(),
	})
	session.markApplyDone()
	s.saveRecordFromSession(session)
	logger.Logger.Infof("AI 诊断修复完成, diagnosisID: %s, status: %s, actions: %v", session.ID, overall, actionIDs(actions))
}

func checkConfirmation(spec *ActionSpec, input ApplyInput, scope *Scope) error {
	switch spec.Confirmation {
	case ConfirmNone:
		return nil
	case ConfirmNormal:
		if !input.Confirm {
			return fmt.Errorf("需要确认")
		}
		return nil
	case ConfirmTyped:
		if !input.Confirm {
			return fmt.Errorf("需要确认")
		}
		expected := ""
		if spec.ID == "update_game" {
			expected = "UPDATE"
		} else if scope != nil && scope.Room != nil {
			expected = scope.Room.GameName
		}
		if expected == "" || input.ConfirmationText != expected {
			return fmt.Errorf("确认文本不匹配")
		}
		return nil
	default:
		return nil
	}
}

func (s *Service) updateActionRecord(record *models.DiagnosisActionRecord, result ActionResult) {
	if record == nil {
		return
	}
	record.Status = result.Status
	record.Code = result.Code
	record.Message = result.Message
	record.Verified = result.Verified
	if result.Status != ActionStatusRunning {
		now := time.Now()
		record.FinishedAt = &now
	}
	if s.diagnosisDao != nil {
		if err := s.diagnosisDao.UpdateActionRecord(record); err != nil {
			logger.Logger.Warnf("更新动作审计记录失败: %v", err)
		}
	}
}

func actionIDs(actions []*Action) []string {
	ids := make([]string, 0, len(actions))
	for _, action := range actions {
		ids = append(ids, action.ActionID)
	}
	return ids
}

func copyParams(params map[string]any) map[string]any {
	out := make(map[string]any, len(params))
	for key, value := range params {
		out[key] = value
	}
	return out
}
