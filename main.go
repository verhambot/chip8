package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/rthornton128/goncurses"
)

const (
	memorySize    = 4096
	registerCount = 16
	stackSize     = 16
	keyCount      = 16
	displayWidth  = 64
	displayHeight = 32
	startAddress  = 0x200
	fontAddress   = 0x50
	fontCount     = 80
)

type Chip8 struct {
	memory     [memorySize]byte
	v          [registerCount]byte
	i          uint16
	pc         uint16
	sp         uint8
	stack      [stackSize]uint16
	delayTimer byte
	soundTimer byte

	display [displayWidth][displayHeight]bool
	draw    bool

	keys [keyCount]bool

	fontSet [fontCount]byte
}

func NewChip8() *Chip8 {
	c := &Chip8{
		fontSet: [80]byte{
			0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
			0x20, 0x60, 0x20, 0x20, 0x70, // 1
			0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
			0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
			0x90, 0x90, 0xF0, 0x10, 0x10, // 4
			0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
			0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
			0xF0, 0x10, 0x20, 0x40, 0x40, // 7
			0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
			0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
			0xF0, 0x90, 0xF0, 0x90, 0x90, // A
			0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
			0xF0, 0x80, 0x80, 0x80, 0xF0, // C
			0xE0, 0x90, 0x90, 0x90, 0xE0, // D
			0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
			0xF0, 0x80, 0xF0, 0x80, 0x80, // F
		},
	}

	for i := 0; i < fontCount; i++ {
		c.memory[fontAddress+i] = c.fontSet[i]
	}

	c.pc = startAddress

	return c
}

func (c *Chip8) LoadROM(filename string) error {
	rom, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if len(rom) > memorySize-startAddress {
		return fmt.Errorf("ROM size is too large: %d bytes (max %d)", len(rom), memorySize-startAddress)
	}

	for i := 0; i < len(rom); i++ {
		c.memory[startAddress+i] = rom[i]
	}

	return nil
}

func (c *Chip8) Cycle() {
	opcode := uint16(c.memory[c.pc])<<8 | uint16(c.memory[c.pc+1])

	c.pc += 2

	switch opcode & 0xF000 {
	case 0x0000:
		switch opcode & 0x00FF {
		case 0x00E0:
			for x := 0; x < displayWidth; x++ {
				for y := 0; y < displayHeight; y++ {
					c.display[x][y] = false
				}
			}
			c.draw = true
		case 0x00EE:
			c.sp--
			c.pc = c.stack[c.sp]
		}
	case 0x1000:
		c.pc = opcode & 0x0FFF
	case 0x2000:
		c.stack[c.sp] = c.pc
		c.sp++
		c.pc = opcode & 0x0FFF
	case 0x3000:
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode & 0x00FF)
		if c.v[x] == kk {
			c.pc += 2
		}
	case 0x4000:
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode & 0x00FF)
		if c.v[x] != kk {
			c.pc += 2
		}
	case 0x5000:
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4
		if c.v[x] == c.v[y] {
			c.pc += 2
		}
	case 0x6000:
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode & 0x00FF)
		c.v[x] = kk
	case 0x7000:
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode & 0x00FF)
		c.v[x] += kk
	case 0x8000:
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4
		switch opcode & 0x000F {
		case 0x0000:
			c.v[x] = c.v[y]
		case 0x0001:
			c.v[x] |= c.v[y]
		case 0x0002:
			c.v[x] &= c.v[y]
		case 0x0003:
			c.v[x] ^= c.v[y]
		case 0x0004:
			sum := uint16(c.v[x]) + uint16(c.v[y])
			if sum > 0xFF {
				c.v[0xF] = 1
			} else {
				c.v[0xF] = 0
			}
			c.v[x] = byte(sum)
		case 0x0005:
			if c.v[x] > c.v[y] {
				c.v[0xF] = 1
			} else {
				c.v[0xF] = 0
			}
			c.v[x] -= c.v[y]
		case 0x0006:
			c.v[0xF] = c.v[x] & 1
			c.v[x] >>= 1
		case 0x0007:
			if c.v[y] > c.v[x] {
				c.v[0xF] = 1
			} else {
				c.v[0xF] = 0
			}
			c.v[x] = c.v[y] - c.v[x]
		case 0x000E:
			c.v[0xF] = (c.v[x] & 0x80) >> 7
			c.v[x] <<= 1
		}
	case 0x9000:
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4
		if c.v[x] != c.v[y] {
			c.pc += 2
		}
	case 0xA000:
		c.i = opcode & 0x0FFF
	case 0xB000:
		c.pc = (opcode & 0x0FFF) + uint16(c.v[0])
	case 0xC000:
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode & 0x00FF)
		c.v[x] = byte(rand.Intn(256)) & kk
	case 0xD000:
		x := uint16(c.v[(opcode&0x0F00)>>8])
		y := uint16(c.v[(opcode&0x00F0)>>4])
		height := opcode & 0x000F

		c.v[0xF] = 0

		for yLine := uint16(0); yLine < height; yLine++ {
			pixel := c.memory[c.i+yLine]
			for xLine := uint16(0); xLine < 8; xLine++ {
				if (pixel & (0x80 >> xLine)) != 0 {
					xPos := (x + xLine) % displayWidth
					yPos := (y + yLine) % displayHeight

					if c.display[xPos][yPos] {
						c.v[0xF] = 1
					}

					c.display[xPos][yPos] = !c.display[xPos][yPos]
				}
			}
		}

		c.draw = true
	case 0xE000:
		x := (opcode & 0x0F00) >> 8
		switch opcode & 0x00FF {
		case 0x009E:
			if c.keys[c.v[x]] {
				c.pc += 2
			}
		case 0x00A1:
			if !c.keys[c.v[x]] {
				c.pc += 2
			}
		}
	case 0xF000:
		x := (opcode & 0x0F00) >> 8
		switch opcode & 0x00FF {
		case 0x0007:
			c.v[x] = c.delayTimer
		case 0x000A:
			keyPressed := false
			for i := 0; i < keyCount; i++ {
				if c.keys[i] {
					c.v[x] = byte(i)
					keyPressed = true
					break
				}
			}
			if !keyPressed {
				c.pc -= 2
			}
		case 0x0015:
			c.delayTimer = c.v[x]
		case 0x0018:
			c.soundTimer = c.v[x]
		case 0x001E:
			if c.i+uint16(c.v[x]) > 0xFFF {
				c.v[0xF] = 1
			} else {
				c.v[0xF] = 0
			}
			c.i += uint16(c.v[x])
		case 0x0029:
			c.i = uint16(fontAddress) + uint16(c.v[x])*5
		case 0x0033:
			c.memory[c.i] = c.v[x] / 100
			c.memory[c.i+1] = (c.v[x] / 10) % 10
			c.memory[c.i+2] = c.v[x] % 10
		case 0x0055:
			for i := uint16(0); i <= x; i++ {
				c.memory[c.i+i] = c.v[i]
			}
			c.i += x + 1
		case 0x0065:
			for i := uint16(0); i <= x; i++ {
				c.v[i] = c.memory[c.i+i]
			}
			c.i += x + 1
		}
	}
}

func (c *Chip8) UpdateTimers() {
	if c.delayTimer > 0 {
		c.delayTimer--
	}

	if c.soundTimer > 0 {
		c.soundTimer--
	}
}

func (c *Chip8) SetKey(key int, pressed bool) {
	c.keys[key] = pressed
}

func main() {
	romPath := flag.String("rom", "", "Path to ROM file")
	cpuFreq := flag.Int("freq", 700, "CPU frequency in Hz")
	flag.Parse()

	if *romPath == "" {
		fmt.Println("Please provide a ROM file using the -rom flag")
		os.Exit(1)
	}

	chip8 := NewChip8()
	err := chip8.LoadROM(*romPath)
	if err != nil {
		log.Fatalf("Failed to load ROM: %v", err)
	}

	stdscr, err := goncurses.Init()
	if err != nil {
		log.Fatalf("Failed to initialize ncurses: %v", err)
	}
	defer goncurses.End()

	goncurses.Cursor(0)
	goncurses.Echo(false)
	goncurses.Raw(true)
	stdscr.Timeout(0)
	stdscr.Keypad(true)

	cycleDelay := time.Second / time.Duration(*cpuFreq)
	timerDelay := time.Second / 60

	keyMap := map[int]int{
		'1': 0x1, '2': 0x2, '3': 0x3, '4': 0xC,
		'q': 0x4, 'w': 0x5, 'e': 0x6, 'r': 0xD,
		'a': 0x7, 's': 0x8, 'd': 0x9, 'f': 0xE,
		'z': 0xA, 'x': 0x0, 'c': 0xB, 'v': 0xF,
	}

	lastTimerUpdate := time.Now()
	for {
		key := stdscr.GetChar()
		if key == 'q'-'a'+1 {
			break
		}

		for i := range chip8.keys {
			chip8.keys[i] = false
		}

		if key != -1 {
			chipKey, ok := keyMap[int(key)]
			if ok {
				chip8.keys[chipKey] = true
			}
		}

		chip8.Cycle()

		if time.Since(lastTimerUpdate) >= timerDelay {
			chip8.UpdateTimers()
			lastTimerUpdate = time.Now()
		}

		if chip8.draw {
			stdscr.Clear()
			for y := 0; y < displayHeight; y++ {
				for x := 0; x < displayWidth; x++ {
					if chip8.display[x][y] {
						stdscr.MoveAddChar(y, x, goncurses.ACS_BLOCK)
					}
				}
			}
			stdscr.Refresh()
			chip8.draw = false
		}

		time.Sleep(cycleDelay)
	}
}
