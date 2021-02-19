package frequency

type fMeta struct {
	word  string
	count int
}

type Frequency struct {
	frqBySec [60]*fMeta
}

func New() *Frequency {
	return &Frequency{}
}

func (h *Frequency) Add(word string) {

}

func (h *Frequency) GetFrequencyByTime(lastNSeconds int) (string, int) {
	return "", 0
}

func (h *Frequency) Clear() {

}
