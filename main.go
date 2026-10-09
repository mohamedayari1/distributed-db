package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

func main() {
	page := make([]byte, 4096)

	fmt.Println(len(page))

	for i :=0 ; i < 7; i++ {
		page[i] = 255
	}
	

	for i :=8 ; i < 15; i++ {
		page[i] = 0x41
	}

	fmt.Print(hex.Dump(page[:16]))
	
	a := make([]byte, 8)
	b := a[2:4]
	b[0] = 0x99
	fmt.Println(a)
	
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:2], 300)
	binary.LittleEndian.PutUint16(buf[2:4], 7)
	fmt.Println("ta3mlo kachta")
	fmt.Println(hex.Dump(buf))
	
	type BNode []byte
	
	const (
		BNODE_NODE = 1
		BNODE_LEAF = 1
	)


	func (node BNode) btype() uint16 {
		node[:2]
	}

	func (node BNode) nkeys() uint16 {
		node[2:4]
	}

	func (node BNode) setHeader(btype uint16, nkeys uint16) {
		binary.LittleEndian.PutUint16(buf[0:2], btype)
		binary.LittleEndian.PutUint16(buf[2:4], nkeys)
	}

	node := BNode([]byte, 4096)
	node.setHeader(BNODE_LEAF, 300)
	fmt.Println(hex.Dump(node[:8]))
	fmt.Println("btype:", node.btype(), "nkeys:", node.nkeys())


















}