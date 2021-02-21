package wordmeta

// metadata for a word
type Data struct {
	Word  string
	Count int
}

func New(w string, c int) *Data {
	return &Data{Word: w, Count: c}
}

type Datas []*Data

// Actually greater
func (ds Datas) Less(i, j int) bool {
	var flag bool
	if ds[i] == nil {
		flag = true
	} else if ds[j] == nil {
		flag = false
	} else {
		flag = ds[i].Count <= ds[j].Count
	}

	return !flag
}

func (ds Datas) Swap(i, j int) {
	ds[i], ds[j] = ds[j], ds[i]
}

func (ds Datas) Len() int {
	return len(ds)
}
