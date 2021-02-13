package wordfilter

import (
	"bufio"
	"cloudcadetest/framework/log"
	"os"
	"unicode/utf8"
)

type Trie struct {
	root *trieNode
}

type trieNode struct {
	children map[rune]*trieNode
	end      bool
}

func NewTrie() *Trie {
	var r = &Trie{
		root: newNode(),
	}
	return r
}

func newNode() *trieNode {
	n := new(trieNode)
	n.children = make(map[rune]*trieNode)
	return n
}

func (t *Trie) InsertFile(path string) {
	f, e := os.Open(path)
	if e != nil {
		log.Warn(e.Error())
		return
	}

	defer func() {
		if e := f.Close(); e != nil {
			log.Error(e.Error())
		}
	}()

	r := bufio.NewReader(f)
	for {
		s, err := r.ReadString('\n')
		if err != nil {
			break
		}
		t.insert(s[:len(s)-1])
	}
}

func (t *Trie) insert(txt string) {
	if len(txt) < 1 {
		return
	}
	node := t.root
	key := []rune(txt)
	for i := 0; i < len(key); i++ {
		if _, exists := node.children[key[i]]; !exists {
			node.children[key[i]] = newNode()
		}
		node = node.children[key[i]]
	}

	node.end = true
}

func (t *Trie) HasDirty(txt string) bool {
	if len(txt) < 1 {
		return false
	}
	node := t.root
	key := []rune(txt)
	var chars []rune
	slen := len(key)
	for i := 0; i < slen; i++ {
		if _, exists := node.children[key[i]]; exists {
			node = node.children[key[i]]
			for j := i + 1; j < slen; j++ {
				if _, exists := node.children[key[j]]; exists {
					node = node.children[key[j]]
					if node.end == true {
						if chars == nil {
							chars = key
						}
						if i <= j {
							return true
						}
						i = j
						node = t.root
						break
					}
				}
			}
			node = t.root
		}
	}
	return false
}

func (t *Trie) Replace(txt string) string {
	if len(txt) < 1 {
		return txt
	}
	node := t.root
	key := []rune(txt)
	var chars []rune = nil
	slen := len(key)
	for i := 0; i < slen; i++ {
		if _, exists := node.children[key[i]]; exists {
			node = node.children[key[i]]
			for j := i + 1; j < slen; j++ {
				if _, exists := node.children[key[j]]; exists {
					node = node.children[key[j]]
					if node.end == true {
						if chars == nil {
							chars = key
						}
						for t := i; t <= j; t++ {
							c, _ := utf8.DecodeRuneInString("*")
							chars[t] = c
						}
						i = j
						node = t.root
						break
					}
				}
			}
			node = t.root
		}
	}
	if chars == nil {
		return txt
	} else {
		return string(chars)
	}
}
