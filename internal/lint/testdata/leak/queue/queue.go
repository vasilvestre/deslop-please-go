package queue

type RunQueue struct {
	Items []int
	Head  int
	limit int
}

func (q *RunQueue) Full() bool { return len(q.Items) >= q.limit && q.Head >= 0 }
