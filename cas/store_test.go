package cas_test

import (
	"context"
	"io"
	"io/ioutil"
	"testing"

	"bazil.org/plop/cas"
	"gocloud.dev/blob"
	"gocloud.dev/blob/memblob"
)

func set(l ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(l))
	for _, s := range l {
		m[s] = struct{}{}
	}
	return m
}

// checkBucket does a sanity check on blobstore contents.
func checkBucket(t testing.TB, bucket *blob.Bucket, want ...string) {
	t.Helper()

	ctx := context.Background()
	iter := bucket.List(nil)
	wantMap := set(want...)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if _, ok := wantMap[obj.Key]; !ok {
			t.Errorf("bucket junk: %+v", obj)
			continue
		}
		delete(wantMap, obj.Key)
	}
	if len(wantMap) > 0 {
		t.Errorf("blobs not seen in bucket: %v", want)
	}
}

func TestRoundtrip(t *testing.T) {
	b := memblob.OpenBucket(nil)
	s := cas.NewStore(b, "s3kr1t")

	// intentionally enforce harsh lifetimes on contexts to make
	// sure we don't remember them too long
	ctxW := context.Background()
	ctxW, cancelW := context.WithCancel(ctxW)
	defer cancelW()
	w := s.Create(ctxW)
	defer w.Abort()
	const greeting = "hello, world\n"
	if _, err := w.Write([]byte(greeting)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	key, err := w.Commit()
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	cancelW()
	t.Logf("created %s", key)

	ctxOpen := context.Background()
	ctxOpen, cancelOpen := context.WithCancel(ctxOpen)
	defer cancelOpen()
	h, err := s.Open(ctxOpen, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	cancelOpen()

	if g, e := h.Size(), int64(len(greeting)); g != e {
		t.Errorf("wrong length: %d != %d", g, e)
	}

	ctxRead := context.Background()
	r := h.IO(ctxRead)
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if g, e := string(buf), greeting; g != e {
		t.Errorf("bad content: %q != %q", g, e)
	}

	checkBucket(t, b,
		"b3jci1t6o4wstq445g5hc6mguexbbq948kq7mm1kxbjwyzwdrh6o",
		"o3iaqfe94q73cqbw3s468pxoy444hotxmahoqkfi91htaigfheqy",
	)
}

func TestCreateSizeZero(t *testing.T) {
	b := memblob.OpenBucket(nil)
	s := cas.NewStore(b, "s3kr1t")

	// intentionally enforce harsh lifetimes on contexts to make
	// sure we don't remember them too long
	ctxW := context.Background()
	ctxW, cancelW := context.WithCancel(ctxW)
	defer cancelW()
	w := s.Create(ctxW)
	defer w.Abort()
	key, err := w.Commit()
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	cancelW()
	t.Logf("created %s", key)

	ctxOpen := context.Background()
	ctxOpen, cancelOpen := context.WithCancel(ctxOpen)
	defer cancelOpen()
	h, err := s.Open(ctxOpen, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	cancelOpen()

	if g, e := h.Size(), int64(0); g != e {
		t.Errorf("wrong length: %d != %d", g, e)
	}

	ctxRead := context.Background()
	r := h.IO(ctxRead)
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if g, e := string(buf), ""; g != e {
		t.Errorf("bad content: %q != %q", g, e)
	}

	checkBucket(t, b,
		"kjbqmr44hxaqeebjd9b9r4dsukrf34ag8kbiacnbg9pd7cpk8t8y",
	)
}

func TestReadAt(t *testing.T) {
	ctx := context.Background()
	b := memblob.OpenBucket(nil)
	s := cas.NewStore(b, "s3kr1t")
	w := s.Create(ctx)
	defer w.Abort()
	const greeting = "hello, world\n"
	if _, err := w.Write([]byte(greeting)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	key, err := w.Commit()
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	h, err := s.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	buf := make([]byte, 3)
	n, err := h.IO(ctx).ReadAt(buf, 4)
	if err != nil {
		t.Fatalf("ReadAt: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("ReadAt returned a weird length: %d", n)
	}
	if g, e := string(buf), greeting[4:4+3]; g != e {
		t.Errorf("bad content: %q != %q", g, e)
	}
}
