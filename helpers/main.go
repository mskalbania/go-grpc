package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"log"
)

func main() {
	payload := int32(268_435_456)
	message := &wrapperspb.Int32Value{Value: payload}
	o, c := compressedSize(message)
	fmt.Printf("original: %d\ncompressed: %d\n", o, c)
}

func compressedSize[M protoreflect.ProtoMessage](msg M) (int, int) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	out, err := proto.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := gz.Write(out); err != nil {
		log.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		log.Fatal(err)
	}
	return len(out), len(b.Bytes())
}
