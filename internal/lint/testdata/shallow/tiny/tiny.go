package tiny

type T struct{ n int }

func (t *T) A() int { t.n++; return t.n }

func (t *T) B() int { t.n--; return t.n }
