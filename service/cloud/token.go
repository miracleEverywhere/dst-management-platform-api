package cloud

import (
	"context"
	"dst-management-platform-api/database/models"
	"time"
)

func (c *Client) accessToken(ctx context.Context, link *models.CloudLink) (string, error) {
	if link.AccessToken == "" || link.RefreshToken == "" {
		return "", ErrRelink
	}
	if link.AccessExpiresAt > time.Now().Unix()+60 {
		return c.decryptToken(link, link.AccessToken)
	}

	return c.refreshLink(ctx, link)
}

func (c *Client) decryptToken(link *models.CloudLink, encrypted string) (string, error) {
	key, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}
	token, err := decryptSecret(key, link.Username, encrypted)
	if err != nil {
		return "", ErrRelink
	}
	if token == "" {
		return "", ErrRelink
	}

	return token, nil
}

func (c *Client) refreshLink(ctx context.Context, link *models.CloudLink) (string, error) {
	if link.RefreshToken == "" {
		return "", ErrRelink
	}
	if link.RefreshExpiresAt > 0 && link.RefreshExpiresAt <= time.Now().Unix() {
		return "", ErrRelink
	}

	refreshToken, err := c.decryptToken(link, link.RefreshToken)
	if err != nil {
		return "", err
	}

	pair, err := c.Refresh(ctx, refreshToken)
	if err != nil {
		if IsUnauthorized(err) || IsForbidden(err) {
			return "", ErrRelink
		}
		return "", err
	}

	if err = applyTokenPair(link, *pair); err != nil {
		return "", err
	}

	return c.decryptToken(link, link.AccessToken)
}

func applyTokenPair(link *models.CloudLink, pair TokenPair) error {
	key, err := loadOrCreateKey()
	if err != nil {
		return err
	}
	access, err := encryptSecret(key, link.Username, pair.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := encryptSecret(key, link.Username, pair.RefreshToken)
	if err != nil {
		return err
	}
	link.AccessToken = access
	link.RefreshToken = refresh
	link.AccessExpiresAt = pair.AccessTokenExpiresAt
	link.RefreshExpiresAt = pair.RefreshTokenExpiresAt

	return nil
}
