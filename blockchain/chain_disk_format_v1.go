package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strings"
)

const chainDiskMagicV1 = "QCHAIN\x00\x01"

// The disk format is independent of the peer wire format. The checksum
// detects accidental corruption. Trusted replay establishes authenticity.
type chainDiskHeaderV1 struct {
	Magic       [8]byte
	Network     [16]byte
	Domain      [32]byte
	Genesis     [32]byte
	ChainID     [32]byte
	PayloadSize uint64
	Checksum    [32]byte
}

func diskHeaderForChainV1(bc *Blockchain) (chainDiskHeaderV1, error) {
	var header chainDiskHeaderV1
	if len(bc.Blocks) == 0 || bc.Blocks[0] == nil || len(bc.Blocks[0].Hash) != sha256.Size {
		return header, fmt.Errorf("chain file: missing genesis hash")
	}
	copy(header.Magic[:], chainDiskMagicV1)
	copy(header.Genesis[:], bc.Blocks[0].Hash)
	if identity := bc.StoredNetworkIdentityV1; identity != nil {
		if len(identity.Network) > len(header.Network) || len(identity.GenesisDomain) > len(header.Domain) ||
			identity.GenesisHash != fmt.Sprintf("%x", header.Genesis) {
			return header, fmt.Errorf("chain file: invalid stored identity")
		}
		copy(header.Network[:], identity.Network)
		copy(header.Domain[:], identity.GenesisDomain)
	}
	header.ChainID = header.chainID()
	return header, nil
}

func (header chainDiskHeaderV1) chainID() [32]byte {
	data := append([]byte("QC_CHAIN_DISK_ID_V1\x00"), header.Network[:]...)
	data = append(data, header.Domain[:]...)
	return sha256.Sum256(append(data, header.Genesis[:]...))
}

func encodeDiskChainV1(writer io.Writer, bc *Blockchain) error {
	file, ok := writer.(io.WriteSeeker)
	if !ok {
		return fmt.Errorf("chain file: seekable writer required")
	}
	header, err := diskHeaderForChainV1(bc)
	if err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, header); err != nil {
		return err
	}
	checksum := sha256.New()
	if err := gob.NewEncoder(io.MultiWriter(file, checksum)).Encode(bc); err != nil {
		return err
	}
	end, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	header.PayloadSize = uint64(end - int64(binary.Size(header)))
	copy(header.Checksum[:], checksum.Sum(nil))
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return binary.Write(file, binary.BigEndian, header)
}

// diskChainPayloadV1 verifies the entire framed payload before Gob decoding.
// An old unframed file is still subject to the trusted canonical replay.
func diskChainPayloadV1(file *os.File) (io.Reader, *chainDiskHeaderV1, error) {
	var magic [8]byte
	n, err := file.ReadAt(magic[:], 0)
	if err != nil && err != io.EOF {
		return nil, nil, err
	}
	if n < 7 || string(magic[:7]) != chainDiskMagicV1[:7] {
		return file, nil, nil
	}
	if n != len(magic) || string(magic[:]) != chainDiskMagicV1 {
		return nil, nil, fmt.Errorf("chain file: unsupported or incomplete format version")
	}
	var header chainDiskHeaderV1
	if err := binary.Read(file, binary.BigEndian, &header); err != nil {
		return nil, nil, fmt.Errorf("chain file: header: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	offset := int64(binary.Size(header))
	size := info.Size() - offset
	if size <= 0 || uint64(size) != header.PayloadSize || header.ChainID != header.chainID() {
		return nil, nil, fmt.Errorf("chain file: invalid payload length or chain identity")
	}
	checksum := sha256.New()
	if _, err := io.Copy(checksum, io.NewSectionReader(file, offset, size)); err != nil {
		return nil, nil, err
	}
	if !bytes.Equal(checksum.Sum(nil), header.Checksum[:]) {
		return nil, nil, fmt.Errorf("chain file: payload checksum mismatch")
	}
	return io.NewSectionReader(file, offset, size), &header, nil
}

func decodeChainGobV1(reader io.Reader, dst any) error {
	decoder := gob.NewDecoder(reader)
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing data after chain")
	}
	return nil
}

func loadDiskChainV1(filename string, trusted *Blockchain) (*Blockchain, error) {
	file, err := openChainFileOSV1(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	payload, header, err := diskChainPayloadV1(file)
	if err != nil {
		return nil, err
	}
	if header != nil && trusted != nil {
		expected, err := diskHeaderForChainV1(trusted)
		untagged := header.matchesStoredIdentity(nil) && (trusted.networkIdentity == nil || trusted.networkIdentity.Network != "mainnet")
		if err != nil || header.Genesis != expected.Genesis || (!untagged && header.ChainID != expected.ChainID) {
			return nil, fmt.Errorf("chain file: header differs from trusted network identity or genesis")
		}
	}
	var loaded Blockchain
	if err := decodeChainGobV1(payload, &loaded); err != nil {
		return nil, fmt.Errorf("chain file: decode: %w", err)
	}
	if header != nil {
		actual, err := diskHeaderForChainV1(&loaded)
		if err != nil || actual.ChainID != header.ChainID {
			return nil, fmt.Errorf("chain file: payload and header identity differ")
		}
	}
	return &loaded, nil
}

func (header chainDiskHeaderV1) matchesStoredIdentity(identity *NetworkIdentityV1) bool {
	if identity == nil {
		return header.Network == [16]byte{} && header.Domain == [32]byte{}
	}
	return strings.TrimRight(string(header.Network[:]), "\x00") == identity.Network &&
		strings.TrimRight(string(header.Domain[:]), "\x00") == identity.GenesisDomain &&
		fmt.Sprintf("%x", header.Genesis) == identity.GenesisHash
}
