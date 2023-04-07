package main

import (
	"fmt"
	"testing"
	"time"
	"unsafe"
)

func Test8BitsRegister(t *testing.T) {
	oam := make([]ObjectAttributeEntry, 64)
	oam[0].y = 10
	oam[0].id = 2
	oam[1].y = 5
	oam[1].id = 10
	pointer := unsafe.Pointer(&oam[0])
	var value *uint8 = ((*uint8)(pointer))
	fmt.Println(*value)
	pointer = unsafe.Add(pointer, 4*unsafe.Sizeof(oam[0].y))
	value = ((*uint8)(pointer))
	fmt.Println(*value)
	pointer = unsafe.Add(pointer, unsafe.Sizeof(oam[0].y))
	value = ((*uint8)(pointer))
	fmt.Println(*value)
}

func TestCreateControlRegister(t *testing.T) {
	control := CreateControlRegister()
	start := time.Now()
	control.GetField("nametable_x")
	elapsed := time.Now().Sub(start)
	fmt.Printf("Time = %s\n", elapsed)

}
