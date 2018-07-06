package compiler

import (
	"fmt"

	"github.com/nitrogen-lang/nitrogen/src/vm/opcode"
)

type Instruction struct {
	Instr     opcode.Opcode
	Args      []uint16 // len = 1 or 2
	ArgLabels []string // len = 1 or 2, name of label for corresponding argument, prefix "~" means relative
	Label     string   // Label names this instruction for linking later
}

func (i *Instruction) String() string {
	return opcode.Names[i.Instr]
}

func (i *Instruction) Is(code opcode.Opcode) bool {
	return i.Instr == code
}

type InstSet []*Instruction

func NewInstSet() InstSet {
	return make(InstSet, 0, 20)
}

func (i InstSet) last() *Instruction {
	return i[len(i)-1]
}

func (i InstSet) addInst(code opcode.Opcode, args ...uint16) {
	checkArgLength(code, len(args))
	i = append(i, &Instruction{
		Instr: code,
		Args:  args,
	})
}

func (i InstSet) addLabel(label string) {
	i = append(i, &Instruction{
		Instr: opcode.Label,
		Label: label,
	})
}

func (i InstSet) addLabeledArgs(code opcode.Opcode, args []uint16, argLabels []string) {
	checkArgLength(code, len(args))
	i = append(i, &Instruction{
		Instr:     code,
		Args:      args,
		ArgLabels: argLabels,
	})
}

func args(a ...uint16) []uint16  { return a }
func argsl(a ...string) []string { return a }

func checkArgLength(code opcode.Opcode, argLen int) {
	if (opcode.HasOneByteArg || opcode.HasTwoByteArg) && argLen != 1 {
		panic(fmt.Sprintf("opcode %s requires 1 16-bit argument, given %d", code.String(), argLen))
	} else if opcode.HasFourByteArg && argLen != 2 {
		panic(fmt.Sprintf("opcode %s requires 2 16-bit argument, given %d", code.String(), argLen))
	}
}

func (i InstSet) Len() int64 {
	size := int64(0)
	for _, i := range i {
		size++

		if opcode.HasOneByteArg[i.Instr] {
			size++
		} else if opcode.HasTwoByteArg[i.Instr] {
			size += 2
		} else if opcode.HasFourByteArg[i.Instr] {
			size += 4
		}
	}
	return size
}

func (i InstSet) Assemble() []byte {
	bytes := make([]byte, i.Len())
	offset := 0

	for _, i := range i {
		if i.Is(opcode.Label) {
			continue
		}

		bytes[offset] = i.Instr.ToByte()
		offset++

		if opcode.HasOneByteArg[i.Instr] {
			bytes[offset] = byte(i.Args[0])
			offset++
		} else if opcode.HasTwoByteArg[i.Instr] {
			arg := uint16ToBytes(i.Args[0])
			bytes[offset] = arg[0]
			bytes[offset+1] = arg[1]
			offset += 2
		} else if opcode.HasFourByteArg[i.Instr] {
			arg := uint16ToBytes(i.Args[0])
			bytes[offset] = arg[0]
			bytes[offset+1] = arg[1]

			arg = uint16ToBytes(i.Args[1])
			bytes[offset+2] = arg[0]
			bytes[offset+3] = arg[1]
			offset += 4
		}
	}

	return bytes
}

type Code struct {
	code []byte
	i    int64
}

func NewCode(code []byte) *Code {
	return &Code{
		code: code,
	}
}

func (c *Code) NextInstruction() *Instruction {
	if c.i >= int64(len(c.code)) {
		return nil
	}

	curr := opcode.Opcode(c.code[c.i])

	i := &Instruction{
		Instr: curr,
		Args:  make([]uint16, 0),
	}
	c.i++

	if opcode.HasOneByteArg[curr] {
		i.Args = []uint16{uint16(c.code[c.i])}
		c.i++
	} else if opcode.HasTwoByteArg[curr] {
		i.Args = []uint16{bytesToUint16(c.code[c.i], c.code[c.i+1])}
		c.i += 2
	} else if opcode.HasFourByteArg[curr] {
		i.Args = []uint16{
			bytesToUint16(c.code[c.i], c.code[c.i+1]),
			bytesToUint16(c.code[c.i+2], c.code[c.i+3]),
		}
		c.i += 4
	}

	return i
}
