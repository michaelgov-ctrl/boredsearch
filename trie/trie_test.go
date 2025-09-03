package trie

/*
	// TODO: make an actual test
for _, s := range []string{
	"refoundation",
	"galravage",
	"antiproductive",
	"unctioneer",
	"Zwinglianist",
} {
	fmt.Println(app.wordTrie.Get(s))
}

walker := func(key string, value any) error {
	fmt.Println(key)
	return nil
}

for _, s := range []string{
	"Zun",
	"kata",
	"AAASA",
} {
	app.wordTrie.WalkLeaves(s, walker)
}

time.Sleep(10 * time.Minute)



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
*/
