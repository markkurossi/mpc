MPCL := ../../../pkg/crypto/sha1

all: diff

diff:
	diff -u crypto/sha1/block.go $(MPCL)/block.mpcl || true
	diff -u crypto/sha1/sha1.go $(MPCL)/sha1.mpcl || true
