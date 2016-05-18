package libkademlia

import (
	"bytes"
	"net"
	"strconv"
	"testing"
	// "sort"
	// "fmt"
	//"time"
)

func TestFindValue(t *testing.T) {
	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	instance := make([]*Kademlia,30)
	host := make([]net.IP, 30)
	port := make([]uint16, 30)
	for i := 30; i < 60; i++ {
		hostnumber := "localhost:"+strconv.Itoa(7900+i)
		instance[i-30] = NewKademlia(hostnumber)
		host[i-30], port[i-30], _ = StringToIpPort(hostnumber)
	}
	for k := 0; k < 29; k++ {
		instance[k].DoPing(host[k+1], port[k+1])
	}
	key := NewRandomID()
	value := []byte("Hello world")
	// err := instance[10].DoStore(&(instance[10].SelfContact), key, value)   //problem of contact???
	// if err != nil {
	// 	t.Error("Could not store value")
	// }
	err := instance[0].DoStore(&(instance[1].SelfContact), key, value)
	if !bytes.Equal(instance[1].HashTable[key],value) {
		t.Error("Stored value in hashtable is not correct")
	}
	// ///// print all stored value
	// for i := 0; i < 30; i++{
	// 	if instance[i].HashTable != nil {
	// 		fmt.Println("stored value:" + instance[i].NodeID.AsString() + String(map["key"]))
	// 	}
	// }
	foundValue, _, err := instance[0].DoFindValue(&(instance[1].SelfContact),key)
	if err != nil {
		t.Error("Error of do find value")
		return
	}
	if foundValue != nil{
		if !bytes.Equal(foundValue, value) {
			t.Error("Stored value did not match found value")
		}
	} else {
		t.Error("Do not find value")
	}
}
