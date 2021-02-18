package game

import (
	"cloudcadetest/framework/module"
)

type FSkeleton int

func NewFS() *FSkeleton {
	return new(FSkeleton)
}

func (s *FSkeleton) GetServerModule() *module.ServerMod {
	return SM
}

func (s *FSkeleton) GetID() int64 {
	return 1
}

func (s *FSkeleton) GetWordListFilePath() string {
	return "list.txt"
}
