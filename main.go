package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"hash/crc32"
	"io"
	"log"
	"os"
	"unicode/utf16"
)

type random struct {
	seed uint32
}

func (r *random) randomNumberGen() uint8 {
	// The encryption of the data is a simple XOR cipher where the key is a number
	// generated by a linear congruential generator.
	// The implementation does not use a modulus.
	r.seed = r.seed*0x19660d + 0x3c6ef35f

	// The channel performs two different left shifts depending on the length of data being XOR'd.
	// We will not be using one of them as it is completely redundant to implement it when we can use one.
	return uint8(r.seed >> 0xc)
}

func main() {
	seed := uint32(1)
	r := random{seed: seed}

	full := new(bytes.Buffer)

	// Magic
	full.WriteString("Yos3")
	temp := new(bytes.Buffer)

	// Decrypted data size.
	err := binary.Write(temp, binary.BigEndian, uint32(328))
	if err != nil {
		log.Fatalln("failed to write decrypted data size")
	}

	err = binary.Write(temp, binary.BigEndian, seed)
	if err != nil {
		log.Fatalln("failed to write seed")
	}

	decrypted := makeUnencryptedData()
	status := 0
	for i := 0; i < len(decrypted); i++ {
		// I don't know why they did it like this, quite possibly to confuse people who would try to RE it?
		if status == 0 {
			// Returning to the aforementioned left shifts on the seed, here is where we ignore it.
			// By writing UINT8_MAX, we can skip one of the shifts entirely as 255 AND any valid value is never 0,
			// which is what performs the shift.
			temp.WriteByte(255)
			status = 128

			// Retain position in decrypted slice.
			i--
			continue
		}

		// Perform the encryption.
		temp.WriteByte(decrypted[i] ^ r.randomNumberGen())
		status >>= 1
	}

	// We must get the MD5 hash of the encrypted data + the seed and 328.
	sum := md5.Sum(temp.Bytes())
	full.Write(sum[:])
	full.Write(temp.Bytes())

	// The end.
	err = os.WriteFile("user.fdu", full.Bytes(), 0666)
	if err != nil {
		log.Fatalln("failed to write file")
	}
}

func makeUnencryptedData() []byte {
	buf := new(bytes.Buffer)

	// CRC32
	write(buf, uint32(0))

	// Value that always must be 0
	write(buf, uint32(0))

	// Instructor name. 12 characters max.
	instructor := utf16.Encode([]rune("WiiLink"))
	instructor = append(instructor, 0)
	write(buf, instructor)

	// Pad to 32 bytes as 4 + 4 + 24
	for buf.Len() != 32 {
		buf.WriteByte(0)
	}

	// Instructor email. Size until EOF (296)
	buf.WriteString("support@wiilink.ca")
	buf.WriteByte(0)

	// Finally pad
	for buf.Len() != 328 {
		buf.WriteByte(0)
	}

	// Calculate and write the CRC32 checksum
	table := crc32.MakeTable(crc32.IEEE)
	checksum := crc32.Checksum(buf.Bytes()[4:328], table)

	binary.BigEndian.PutUint32(buf.Bytes(), checksum)
	return buf.Bytes()
}

func write(buf io.Writer, data any) {
	err := binary.Write(buf, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}
}