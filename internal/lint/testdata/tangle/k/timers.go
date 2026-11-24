package k

type Timers struct{ tasks *Tasks }

func (t *Timers) Fire() int {
	if t.tasks == nil {
		return 0
	}
	return t.tasks.Spawn()
}
