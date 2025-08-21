package main

import (
	"bufio"
	"fmt"
	"os"
	"unicode/utf8"
)

/*
	much of this is pulled from:
	https://github.com/dghubble/trie/blob/main/rune_trie.go
*/

// Trie is a trie of runes with string keys and any values.
// Note that internal nodes have nil values so a stored nil value will not
// be distinguishable and will not be included in Walks.
type Trie struct {
	value    any
	children map[rune]*Trie
}

// NewTrie allocates and returns a new *Trie.
func NewTrie() *Trie {
	return new(Trie)
}

func (t *Trie) LoadFromFile(file *os.File) (*Trie, error) {
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t.Put(scanner.Text(), 1)
	}
	if err := scanner.Err(); err != nil {
		return t, err
	}

	return t, nil
}

// Get returns the value stored at the given key. Returns nil for internal
// nodes or for nodes with a value of nil.
func (trie *Trie) Get(key string) any {
	node := trie
	for _, r := range key {
		node = node.children[r]
		if node == nil {
			return nil
		}
	}
	return node.value
}

// Put inserts the value into the trie at the given key, replacing any
// existing items. It returns true if the put adds a new value, false
// if it replaces an existing value.
// Note that internal nodes have nil values so a stored nil value will not
// be distinguishable and will not be included in Walks.
func (trie *Trie) Put(key string, value any) bool {
	node := trie
	for _, r := range key {
		child := node.children[r]
		if child == nil {
			if node.children == nil {
				node.children = map[rune]*Trie{}
			}
			child = new(Trie)
			node.children[r] = child
		}
		node = child
	}
	// does node have an existing value?
	isNewVal := node.value == nil
	node.value = value
	return isNewVal
}

// Delete removes the value associated with the given key. Returns true if a
// node was found for the given key. If the node or any of its ancestors
// becomes childless as a result, it is removed from the trie.
func (trie *Trie) Delete(key string) bool {
	path := make([]nodeRune, len(key)) // record ancestors to check later
	node := trie
	for i, r := range key {
		path[i] = nodeRune{r: r, node: node}
		node = node.children[r]
		if node == nil {
			// node does not exist
			return false
		}
	}
	// delete the node value
	node.value = nil
	// if leaf, remove it from its parent's children map. Repeat for ancestor
	// path.
	if node.isLeaf() {
		// iterate backwards over path
		for i := len(key) - 1; i >= 0; i-- {
			if path[i].node == nil {
				continue
			}
			parent := path[i].node
			r := path[i].r
			delete(parent.children, r)
			if !parent.isLeaf() {
				// parent has other children, stop
				break
			}
			parent.children = nil
			if parent.value != nil {
				// parent has a value, stop
				break
			}
		}
	}
	return true // node (internal or not) existed and its value was nil'd
}

// WalkFunc defines some action to take on the given key and value during
// a Trie Walk. Returning a non-nil error will terminate the Walk.
type WalkFunc func(key string, value any) error

// Walk iterates over each key/value stored in the trie and calls the given
// walker function with the key and value. If the walker function returns
// an error, the walk is aborted.
// The traversal is depth first with no guaranteed order.
func (trie *Trie) Walk(walker WalkFunc) error {
	return trie.walk("", walker)
}

// WalkLeaves iterates over each key/value of every node after the given prefix,
// calling the given walker function for each key/value.
// If the walker function returns an error, the walk is aborted.
func (trie *Trie) WalkLeaves(prefix string, walker WalkFunc) error {
	node := trie
	for _, r := range prefix {
		next, ok := node.children[r]
		if !ok || next == nil {
			return nil // not found, nothing to walk. consider stepping up the trie & retrying based on popularity
		}
		node = next
	}

	if node.value != nil {
		if err := walker(prefix, node.value); err != nil {
			return err
		}
	}

	for r, child := range node.children {
		// +string() could be optimized
		if err := child.walk(prefix+string(r), walker); err != nil {
			return err
		}
	}

	return nil
}

// WalkPath iterates over each key/value in the path in trie from the root to
// the node at the given key, calling the given walker function for each
// key/value. If the walker function returns an error, the walk is aborted.
func (trie *Trie) WalkPath(key string, walker WalkFunc) error {
	// Get root value if one exists.
	if trie.value != nil {
		if err := walker("", trie.value); err != nil {
			return err
		}
	}

	for i, r := range key {
		if trie = trie.children[r]; trie == nil {
			return nil
		}
		if trie.value != nil {
			end := i + utf8.RuneLen(r)
			if err := walker(string(key[:end]), trie.value); err != nil {
				return err
			}
		}
	}
	return nil
}

// Trie node and the rune key of the child the path descends into.
type nodeRune struct {
	node *Trie
	r    rune
}

func (trie *Trie) walk(key string, walker WalkFunc) error {
	if trie.value != nil {
		if err := walker(key, trie.value); err != nil {
			return err
		}
	}

	for r, child := range trie.children {
		// +string() could be optimized
		if err := child.walk(key+string(r), walker); err != nil {
			return err
		}
	}

	return nil
}

func (trie *Trie) isLeaf() bool {
	return len(trie.children) == 0
}

// TODO: break this out to an actual test
func (trie Trie) TestTrie() error {
	trie = *NewTrie()

	tests := []struct {
		key   string
		value any
	}{
		{"t", -1},
		{"test", 0},
		{"tests", 1},
		{"testing", 2},
		{"testin", 3},
		{"test", 5},
	}

	for _, t := range tests {
		if isNew := trie.Put(t.key, t.value); !isNew {
			fmt.Println(t.key, "----", trie.Get(t.key))
		}
	}

	walker := func(key string, value any) error {
		fmt.Println(key)
		return nil
	}

	fmt.Printf("\nbefore\n")
	if err := trie.WalkPath("testi", walker); err != nil {
		return err
	}

	fmt.Printf("\nafter\n")
	if err := trie.WalkLeaves("testi", walker); err != nil {
		return err
	}

	fmt.Printf("\nall\n")
	if err := trie.Walk(walker); err != nil {
		return err
	}

	return nil
}
