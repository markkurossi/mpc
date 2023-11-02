
MPCL := ../../../pkg/crypto/curve25519

all: diff

diff:
	diff -u curve25519/doc.go $(MPCL)/doc.mpcl || true
	diff -u curve25519/curve25519.go $(MPCL)/curve25519.mpcl || true
