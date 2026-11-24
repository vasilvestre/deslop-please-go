package k

type Tasks struct{ timers *Timers }

func (t *Tasks) Spawn() int {
	t.timers.Fire()
	return 1
}
