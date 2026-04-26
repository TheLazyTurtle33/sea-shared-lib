package cleanup

type Cleaner interface {
	Clean()
}

var Cleaners []Cleaner

func RegisterCleaner(c Cleaner) {
	Cleaners = append(Cleaners, c)
}

func Clean() {
	for _, c := range Cleaners {
		c.Clean()
	}
}
