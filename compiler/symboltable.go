package compiler

import "fmt"

type tableItem struct {
	name     string
	ttype    string
	kind     string
	position int
}

type table struct {
	items          map[string]tableItem
	segmentCounter map[string]int
}

type symbolTable struct {
	currentTbl int
	tbl        []table
}

func newSymbolTable() *symbolTable {
	tbl := &symbolTable{
		currentTbl: -1,
		tbl:        make([]table, 100),
	}
	tbl.next()
	return tbl
}

func (s *symbolTable) kindOf(name string) string {
	item, ok := s.find(name)
	if ok {
		return item.kind
	}
	return ""
}

func (s *symbolTable) typeOf(name string) string {
	item, ok := s.find(name)
	if ok {
		return item.ttype
	}
	return ""
}

func (s *symbolTable) indexOf(name string) int {
	item, ok := s.find(name)
	if ok {
		return item.position
	}
	return 0
}

func (s *symbolTable) debug() {

	cnt := s.currentTbl
	fmt.Printf(" ------------------------------------------------------------------------\n")
	fmt.Printf("| %-70s |\n", fmt.Sprintf("LVL[%d]", cnt))
	fmt.Printf(" ------------------------------------------------------------------------\n")
	fmt.Printf("| %-20s | %-20s | %-20s | # | \n", "NAME", "TYPE", "KIND")
	for cnt >= 0 {
		tbl := s.tbl[cnt]
		for _, val := range tbl.items {
			fmt.Printf("| %-20s | %-20s | %-20s | %d |\n", val.name, val.ttype, val.kind, val.position)
		}
		cnt--
	}
	fmt.Printf(" ------------------------------------------------------------------------\n\n")
}

func (s *symbolTable) varCount(kind string) int {
	total := 0
	cnt := s.currentTbl
	for cnt >= 0 {
		total += s.tbl[cnt].segmentCounter[kind]
		cnt--
	}
	return total
}

func (s *symbolTable) find(name string) (tableItem, bool) {
	cnt := s.currentTbl
	for cnt >= 0 {
		if item, ok := s.tbl[cnt].items[name]; ok && item.name == name {
			return item, true
		}
		cnt--
	}
	// not found
	return tableItem{}, false
}

func (s *symbolTable) next() {
	s.currentTbl++
	// reset before use
	s.tbl[s.currentTbl] = table{
		items:          make(map[string]tableItem),
		segmentCounter: make(map[string]int),
	}
}

func (s *symbolTable) prev() {
	s.tbl[s.currentTbl].items = make(map[string]tableItem)
	s.tbl[s.currentTbl].segmentCounter = make(map[string]int)
	s.currentTbl--
}

func (s *symbolTable) define(name, ttype, kind string) {
	if kind == "field" {
		kind = "this"
	}
	s.tbl[s.currentTbl].items[name] = tableItem{
		name:     name,
		ttype:    ttype,
		kind:     kind,
		position: s.tbl[s.currentTbl].segmentCounter[kind],
	}
	s.tbl[s.currentTbl].segmentCounter[kind]++
}
