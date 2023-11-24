MPCL := ../../../pkg/crypto/ed25519/internal/edwards25519

all: diff

diff:
	diff -u edwards25519/const.go $(MPCL)/const.mpcl || true
	diff -u edwards25519/edwards25519.go $(MPCL)/ed25519.mpcl || true
