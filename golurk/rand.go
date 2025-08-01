package golurk

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"math/rand/v2"
)

var (
	internalSeed = CreateRandomStateSeed()
	internalRng  = CreateRNG(&internalSeed)
)

func CreateRandomStateSeed() rand.PCG {
	var randBytes [16]byte
	_, err := cryptoRand.Read(randBytes[:])
	if err != nil {
		// Is this smart? Probably not. However in this case I really have no clue how it could error
		panic(err)
	}

	return *rand.NewPCG(binary.LittleEndian.Uint64(randBytes[0:8]), binary.LittleEndian.Uint64(randBytes[8:]))
}

func CreateRNG(seed *rand.PCG) *rand.Rand {
	return rand.New(seed)
}
