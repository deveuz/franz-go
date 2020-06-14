package kgo

import (
	"bytes"
	"sync"
	"testing"
)

func TestNewCompressor(t *testing.T) {
	for i, test := range []struct {
		codecs []CompressionCodec
		fail   bool
	}{
		{codecs: []CompressionCodec{{codec: -1}}, fail: true},
		{codecs: []CompressionCodec{{codec: 5}}, fail: true},

		{codecs: []CompressionCodec{{codec: 0}}},
		{codecs: []CompressionCodec{{codec: 1}, {codec: 0}}},
		{codecs: []CompressionCodec{{codec: 2}, {codec: 0}}},
		{codecs: []CompressionCodec{{codec: 3}}},
		{codecs: []CompressionCodec{{codec: 4}}},
		{codecs: []CompressionCodec{{codec: 4}, {codec: 3}}},

		{codecs: []CompressionCodec{{codec: 1, level: 127}}}, // bad gzip level is defaulted fine
		{codecs: []CompressionCodec{{codec: 4, level: 127}}}, // bad zstd level, same

		{codecs: []CompressionCodec{
			{codec: 4},
			{codec: 4},
			{codec: 3},
			{codec: 2},
			{codec: 1, level: 1},
		}},
	} {
		_, err := newCompressor(test.codecs...)
		fail := err != nil
		if fail != test.fail {
			t.Errorf("#%d: ok? %v, exp ok? %v", i, !fail, !test.fail)
		}
	}
}

func TestCompressDecompress(t *testing.T) {
	d := newDecompressor()
	in := []byte("foo")
	for _, produceVersion := range []int16{
		0, 7,
	} {
		for _, codecs := range [][]CompressionCodec{
			[]CompressionCodec{{codec: 0}},
			[]CompressionCodec{{codec: 1}},
			[]CompressionCodec{{codec: 2}},
			[]CompressionCodec{{codec: 3}},
			[]CompressionCodec{{codec: 4}},
			[]CompressionCodec{{codec: 4}, {codec: 3}},
		} {
			c, _ := newCompressor(codecs...)
			if c == nil {
				if codecs[0].codec == 0 {
					continue
				}
				t.Errorf("unexpected nil compressor from codecs %v", codecs)
			}
			var wg sync.WaitGroup
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					w := sliceWriters.Get().(*sliceWriter)
					defer sliceWriters.Put(w)
					got, used := c.compress(w, in, produceVersion)

					got, err := d.decompress(got, byte(used))
					if err != nil {
						t.Errorf("unexpected decompress err: %v", err)
						return
					}
					if !bytes.Equal(got, in) {
						t.Errorf("got decompress %s != exp compress in %s", got, in)
					}

				}()
			}
			wg.Wait()
		}
	}
}

func BenchmarkCompress(b *testing.B) {
	c, _ := newCompressor(CompressionCodec{codec: 2}) // snappy
	in := []byte("foo")
	for i := 0; i < b.N; i++ {
		w := sliceWriters.Get().(*sliceWriter)
		c.compress(w, in, 0)
		sliceWriters.Put(w)
	}
}
