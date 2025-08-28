package main

import (
	"bufio"
	"errors"
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
	keys     []rune // TODO: on insert add keys for each node to use for ordered retrieval in walk()
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
func (t *Trie) Put(key string, value any) bool {
	node := t
	for _, r := range key {
		if node.children == nil {
			node.children = make(map[rune]*Trie)
		}
		child := node.children[r]
		if child == nil {
			child = new(Trie)
			node.children[r] = child

			node.keys = append(node.keys, r)
		}
		node = child
	}
	isNew := node.value == nil
	node.value = value
	return isNew
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

	for _, r := range node.keys {
		child := node.children[r]
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

	for _, r := range trie.keys {
		child := trie.children[r]
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

var errPageFull = errors.New("page full")

// WalkLeavesWindow walks leaves under prefix in the order defined by node.keys.
// It skips the first `skip` leaves, emits up to `window` leaves, and then stops.
// It returns (emitted, more, err) where `more` is true if additional leaves exist.
func (t *Trie) WalkLeavesWindow(prefix string, skip, window int, walker WalkFunc) (int, bool, error) {
	node := t
	for _, r := range prefix {
		next, ok := node.children[r]
		if !ok || next == nil {
			return 0, false, nil // not found, nothing to walk. consider stepping up the trie & retrying based on popularity
		}
		node = next
	}

	var (
		path             = []rune(prefix)
		visited, emitted int
		more             bool
	)

	var dfs func(n *Trie) error

	dfs = func(n *Trie) error {
		if n.value != nil {
			switch {
			case visited < skip:
				visited++
			case emitted < window:
				if err := walker(string(path), n.value); err != nil {
					return err
				}
				emitted++
			default:
				more = true
				return errPageFull
			}
		}

		for _, r := range n.keys {
			if emitted >= window {
				more = true
				return errPageFull
			}

			path = append(path, r)

			if err := dfs(n.children[r]); err != nil {
				return err
			}

			path = path[:len(path)-1]
		}

		return nil
	}

	err := dfs(node)
	if errors.Is(err, errPageFull) {
		err = nil
	}

	return emitted, more, err
}
