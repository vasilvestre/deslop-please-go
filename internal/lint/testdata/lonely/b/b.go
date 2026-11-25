package b

type Open interface{ Do() }

type One struct{}

func (One) Do() {}

type Two struct{}

func (Two) Do() {}
