// SPDX-FileCopyrightText: © 2023 OneEyeFPV oneeyefpv@gmail.com
// SPDX-License-Identifier: GPL-3.0-or-later
// SPDX-License-Identifier: FS-0.9-or-later

package devices

/*
#include <fcntl.h>
#include <sys/mman.h>
#include <unistd.h>
#include <sys/stat.h>

int createSharedMemory(char *name, int size) {
    int fd = shm_open(name, O_CREAT|O_RDWR, 0666);
    if (fd < 0) {
        return -1; // failed to open
    }
    int errno = ftruncate(fd, size);
    return fd;
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"syscall"
	"unsafe"

	"github.com/kaack/elrs-joystick-control/pkg/util"
	"github.com/veandco/go-sdl2/sdl"
)

type InputGamepad struct {
	Id   string `json:"id"`
	Name string `json:"name"`

	Joy             *sdl.Joystick `json:"-"`
	sharedMemAxis   *[6]int32     `json:"-"`
	sharedMemButton *[6]int32     `json:"-"`
}

func (d *InputGamepad) Axis(axis int) util.RawValue {
	// return util.RawValue(d.Joy.Axis(axis))
	print("Axis [", axis, "] = ", d.sharedMemAxis[axis], "\n")
	return util.RawValue(d.sharedMemAxis[axis])
}

func (d *InputGamepad) Button(button int) util.RawValue {
	if button <= 5 {
		print("Button [", button, "] = ", d.sharedMemButton[button], "\n")
		return util.RawValue(d.sharedMemButton[button])
	} else {
		return util.RawValue(d.Joy.Button(button))
	}
}

func (d *InputGamepad) Hat(hat int) util.RawValue {
	return util.MapRange(util.RawValue(d.Joy.Hat(hat)), -1, 1, util.MinRaw, util.MaxRaw)
}

func (d *InputGamepad) Close() {
	d.Joy.Close()
}

func (d *InputGamepad) InstanceId() int32 {
	return int32(d.Joy.InstanceID())
}
func (d *InputGamepad) Axes() int32 {
	return int32(d.Joy.NumAxes())
}

func (d *InputGamepad) Buttons() int32 {
	return int32(d.Joy.NumButtons())
}

func (d *InputGamepad) Hats() int32 {
	return int32(d.Joy.NumHats())
}

func NewDevice(joy *sdl.Joystick) InputGamepad {

	const size = 64 // size of an 6 integer in bytes
	shmName := C.CString("/myshm")
	// defer C.free(unsafe.Pointer(shmName))

	fd := int(C.createSharedMemory(shmName, C.int(size)))
	if fd < 0 {
		log.Fatal("Failed to create and open shared memory")
	}
	// defer C.close(C.int(fd))

	// Memory map the shared memory object
	addr, err := syscall.Mmap(fd, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		log.Fatal("Error memory-mapping the file:", err)
	}
	// defer syscall.Munmap(addr)

	// Create a slice backed by shared memory
	s := (*[16]int32)(unsafe.Pointer(&addr[0])) // using int32 assuming a 4-byte int

	for i := range s {
		s[i] = int32(100 + i) // Setting the values in the shared memory
	}
	fmt.Println("Shared memory initialized with values:", s)

	// s1 = the first 6 elements of the shared memory
	s1 := (*[6]int32)(unsafe.Pointer(&addr[0]))
	// s2 = the rest of the elements of the shared memory (16-6=10 elements)
	s2 := (*[6]int32)(unsafe.Pointer(&addr[6*4]))

	return InputGamepad{
		Id:              GetJoyStickId(joy),
		Name:            joy.Name(),
		Joy:             joy,
		sharedMemAxis:   s1,
		sharedMemButton: s2,
	}
}

type FakeInputGamepad InputGamepad

func (d *InputGamepad) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		FakeInputGamepad
		Axes    int32 `json:"axes"`
		Buttons int32 `json:"buttons"`
		Hats    int32 `json:"hats"`
	}{
		FakeInputGamepad: FakeInputGamepad(*d),
		Axes:             d.Axes(),
		Buttons:          d.Buttons(),
		Hats:             d.Hats(),
	})
}
