package scheduler

var jobs = []JobConfig{
	{
		Name:     "test",
		Func:     printTest,
		Args:     []interface{}{3},
		TimeType: "second",
		Interval: 30,
		DayAt:    "",
	},
}
