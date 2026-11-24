package store

type Store struct{ n int }

func (s *Store) helper0() int { s.n++; return s.n }

func (s *Store) helper1() int { s.n--; return s.n }

func (s *Store) M0() int {
	s.n += 0
	return s.helper0()
}

func (s *Store) M1() int {
	s.n += 1
	return s.helper1()
}

func (s *Store) M2() int {
	s.n += 2
	return s.helper0()
}

func (s *Store) M3() int {
	s.n += 3
	return s.helper1()
}

func (s *Store) M4() int {
	s.n += 4
	return s.helper0()
}

func (s *Store) M5() int {
	s.n += 5
	return s.helper1()
}

func (s *Store) M6() int {
	s.n += 6
	return s.helper0()
}

func (s *Store) M7() int {
	s.n += 7
	return s.helper1()
}
