package game

import (
	"cloudcadetest/framework/module"
)

type filterSkeleton int

func NewFS() *filterSkeleton {
	return new(filterSkeleton)
}

func (fs *filterSkeleton) GetServerModule() *module.ServerMod {
	return SM
}

func (fs *filterSkeleton) GetID() int64 {
	return 1
}

func (fs *filterSkeleton) GetWordListFilePath() string {
	return "list.txt"
}
