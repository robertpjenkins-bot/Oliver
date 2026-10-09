package design

import (
	"crypto/rand"
	"crypto/sha512"
)

// HashPassword returns a SHA-512 crypt hash ("$6$...") of password, the format
// /etc/shadow on Arch expects. Only the hash is stored in a design, so a saved
// .osmaker file never contains the password itself.
func HashPassword(password string) string {
	const saltChars = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		panic(err) // crypto/rand does not fail on Linux
	}
	salt := make([]byte, 16)
	for i, b := range raw {
		salt[i] = saltChars[int(b)%len(saltChars)]
	}
	return sha512Crypt(password, string(salt))
}

// sha512Crypt implements Ulrich Drepper's SHA-crypt with the default 5000
// rounds, as glibc's crypt(3) does for "$6$" salts.
func sha512Crypt(password, salt string) string {
	if len(salt) > 16 {
		salt = salt[:16]
	}
	p, s := []byte(password), []byte(salt)
	const rounds = 5000

	b := sha512.New()
	b.Write(p)
	b.Write(s)
	b.Write(p)
	db := b.Sum(nil)

	a := sha512.New()
	a.Write(p)
	a.Write(s)
	i := len(p)
	for ; i > 64; i -= 64 {
		a.Write(db)
	}
	a.Write(db[:i])
	for i := len(p); i > 0; i >>= 1 {
		if i&1 != 0 {
			a.Write(db)
		} else {
			a.Write(p)
		}
	}
	da := a.Sum(nil)

	dp := sha512.New()
	for range len(p) {
		dp.Write(p)
	}
	pseq := repeat(dp.Sum(nil), len(p))

	ds := sha512.New()
	for range 16 + int(da[0]) {
		ds.Write(s)
	}
	sseq := repeat(ds.Sum(nil), len(s))

	c := da
	for r := range rounds {
		h := sha512.New()
		if r&1 != 0 {
			h.Write(pseq)
		} else {
			h.Write(c)
		}
		if r%3 != 0 {
			h.Write(sseq)
		}
		if r%7 != 0 {
			h.Write(pseq)
		}
		if r&1 != 0 {
			h.Write(c)
		} else {
			h.Write(pseq)
		}
		c = h.Sum(nil)
	}

	out := []byte("$6$" + salt + "$")
	order := [][3]int{
		{0, 21, 42}, {22, 43, 1}, {44, 2, 23}, {3, 24, 45}, {25, 46, 4},
		{47, 5, 26}, {6, 27, 48}, {28, 49, 7}, {50, 8, 29}, {9, 30, 51},
		{31, 52, 10}, {53, 11, 32}, {12, 33, 54}, {34, 55, 13}, {56, 14, 35},
		{15, 36, 57}, {37, 58, 16}, {59, 17, 38}, {18, 39, 60}, {40, 61, 19},
		{62, 20, 41},
	}
	for _, o := range order {
		out = b64From24(out, c[o[0]], c[o[1]], c[o[2]], 4)
	}
	out = b64From24(out, 0, 0, c[63], 2)
	return string(out)
}

func repeat(digest []byte, n int) []byte {
	out := make([]byte, 0, n)
	for len(out) < n {
		out = append(out, digest[:min(len(digest), n-len(out))]...)
	}
	return out
}

func b64From24(out []byte, b2, b1, b0 byte, n int) []byte {
	const itoa64 = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	w := uint(b2)<<16 | uint(b1)<<8 | uint(b0)
	for range n {
		out = append(out, itoa64[w&0x3f])
		w >>= 6
	}
	return out
}
