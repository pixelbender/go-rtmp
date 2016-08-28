# Golang: RTMP Protocol

## Features

- [x] AMF0 Encoder/Decoder
- [ ] AMF3 Encoder/Decoder
- [ ] FLV Reader/Writer
- [x] RTMP Client
- [ ] RTMP Server

## Installation

```sh
go get github.com/pixelbender/go-rtmp
```

## RTMP Client

```go
package main

import (
    "github.com/pixelbender/go-rtmp/rtmp"
    "fmt"
)

func main() {
    conn, err := rtmp.Dial("rtmp://example.org/app")
    if err != nil {
        fmt.Println(err)
    } else {
        defer conn.Close()
        fmt.Println(conn)
    }
}
```

## Specifications

- [AMF0: Action Message Format](http://wwwimages.adobe.com/content/dam/Adobe/en/devnet/amf/pdf/amf0-file-format-specification.pdf)
- [AMF3: Action Message Format](http://wwwimages.adobe.com/www.adobe.com/content/dam/Adobe/en/devnet/amf/pdf/amf-file-format-spec.pdf)
- [FLV: Video File Format Specification v10](https://www.adobe.com/content/dam/Adobe/en/devnet/flv/pdfs/video_file_format_spec_v10.pdf)
- [FLV: Adobe Flash Video File Format Specification v10.1](http://download.macromedia.com/f4v/video_file_format_spec_v10_1.pdf)
- [RTMP: Real-Time Messaging Protocol](https://www.adobe.com/content/dam/Adobe/en/devnet/rtmp/pdf/rtmp_specification_1.0.pdf)
- [RTMFP: Real-Time Media Flow Protocol](https://tools.ietf.org/html/rfc7016)
- [RTMFP Profile for Flash Communication](https://tools.ietf.org/html/rfc7425)
