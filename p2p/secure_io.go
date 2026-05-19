// p2p/secure_io.go
package p2p

import (
	"bufio"
	"errors"
	"io"
	"net"
	"time"
)

const (
	// Okuma / yazma için timeout
	readTimeout  = 15 * time.Second
	writeTimeout = 15 * time.Second
)

var (
	// Saldırgan veya yanlış davranan peer'lerin tespiti için hata
	ErrMessageTooLarge = errors.New("p2p: message too large")
)

// SafeReader: net.Conn üstüne wrap edilmiş, limitli okuma yapan reader
type SafeReader struct {
	Conn   net.Conn
	Reader *bufio.Reader
}

// NewSafeReader: Her connection için bir kere oluştur
func NewSafeReader(conn net.Conn) *SafeReader {
	return &SafeReader{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
	}
}

// ReadLineLimited: \n ile biten bir mesajı, MaxMessageSize sınırı ile okur.
// Şu an protokolün gob bazlı olduğu için bu helper'ı kullanmıyoruz; ileride
// satır bazlı bir protokole geçersen hazır dursun diye bırakıyoruz.
func (sr *SafeReader) ReadLineLimited() ([]byte, error) {
	if err := sr.Conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return nil, err
	}

	line, err := sr.Reader.ReadBytes('\n')

	if len(line) > MaxMessageSize {
		// Çok büyük mesaj -> DoS kabul etme
		return nil, ErrMessageTooLarge
	}

	// io.EOF vs. diğer hatalar yukarıya aynen döndürülür
	return line, err
}

// WriteLimited: Mesajı boyut kontrolü ve timeout ile yazar
func WriteLimited(conn net.Conn, data []byte) error {
	if len(data) > MaxMessageSize {
		return ErrMessageTooLarge
	}

	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}

	n, err := conn.Write(data)
	if err != nil {
		return err
	}
	if n < len(data) {
		return io.ErrShortWrite
	}

	return nil
}
