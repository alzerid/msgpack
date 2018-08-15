package msgpack
import (
    "testing"
    "strings"
    "reflect"
    "bytes"
    "log"
    "fmt"
)

// Function generates chars into a string build from 0x31 to 0x7e
func generateChar(sb *strings.Builder, max int) {
    for i:=0; i<max; i++ {
        sb.WriteByte(byte(i+0x31))
    }
}

func encodeDebug(t *testing.T, enc *Encoder, buf *bytes.Buffer, v interface{}) {
    enc.Encode(v)
    t.Logf("Encoded[%d]: \"0x%x\"\n", buf.Len(), buf.Bytes())
}

// Function decodes once
func decodeDebug(t *testing.T, dec *Decoder, v interface{}) {

    //Get token
    tok, err := dec.Token()
    if err != nil {
        panic(err)
    }

    //Token not equal to value
    log.Println(v, tok, reflect.TypeOf(v), reflect.TypeOf(tok))
    if !reflect.DeepEqual(v, tok) {
        log.Panicf("value mismatch (%v)! %v (%v) != %v (%v)", dec.Kind(),  v, reflect.TypeOf(v), tok, reflect.TypeOf(tok))
    }

    t.Logf("Checked out! [%v] %v (%v) == %v (%v)", dec.Kind(), v, reflect.TypeOf(v), tok, reflect.TypeOf(tok))
}

func TestInt(t *testing.T) {
    bknown := []byte{ 0xe3, 0xd1, 0x40, 0x74, 0xd2, 0x00, 0x10, 0x00,
                      0x00, 0xd3, 0x00, 0x33, 0xff, 0xaa, 0xbb, 0xcc,
                      0xee, 0xff, 0xd3, 0x00, 0x33, 0xff, 0xaa, 0xbb,
                      0xcc, 0xee, 0xff }

    //Native int types
    fixint := int8(-3)
    integ16 := int16(16500)
    integ32 := int32(1<<20)
    integ64 := int64(0x0033ffaabbcceeff)
    integ := int(0x0033ffaabbcceeff)

    //Encode the above native types
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, fixint)
    encodeDebug(t, enc, &buf, integ16)
    encodeDebug(t, enc, &buf, integ32)
    encodeDebug(t, enc, &buf, integ64)
    encodeDebug(t, enc, &buf, integ)

    //Check if bytes are same
    if bytes.Compare(buf.Bytes(), bknown) != 0 {
        panic("Bytes mismatch!")
    }

    //Decode
    dec := NewDecoder(&buf)

    //Check if we encoded correctly
    decodeDebug(t, dec, fixint)
    decodeDebug(t, dec, integ16)
    decodeDebug(t, dec, integ32)
    decodeDebug(t, dec, integ64)
}

func TestUint(t *testing.T) {
    bknown := []byte{ 0xcc, 0xff, 0xcd, 0x40, 0x74, 0xce, 0x00, 0x10,
                      0x00, 0x00, 0xcf, 0x00, 0x33, 0xff, 0xaa, 0xbb,
                      0xcc, 0xee, 0xff, 0xcf, 0x00, 0x33, 0xff, 0xaa,
                      0xbb, 0xcc, 0xee, 0xff }

    //Native types
    integ8 := uint8(255)
    integ16 := uint16(16500)
    integ32 := uint32(1<<20)
    integ64 := uint64(0x0033ffaabbcceeff)
    integ := uint(0x0033ffaabbcceeff)

    //Encode the above native types
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, integ8)
    encodeDebug(t, enc, &buf, integ16)
    encodeDebug(t, enc, &buf, integ32)
    encodeDebug(t, enc, &buf, integ64)
    encodeDebug(t, enc, &buf, integ)

    // Check if bytes are same
    if !bytes.Equal(buf.Bytes(), bknown) {
        panic("Bytes mismatch!")
    }

    //Check if we encoded correctly
    dec := NewDecoder(&buf)
    decodeDebug(t, dec, integ8)
    decodeDebug(t, dec, integ16)
    decodeDebug(t, dec, integ32)
    decodeDebug(t, dec, integ64)
}

func TestBoolEncoder(t *testing.T) {
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, true)
    encodeDebug(t, enc, &buf, false)
    if bytes.Compare(buf.Bytes(), []byte{ 0xc3, 0xc2 }) != 0 {
        panic("Bytes mismatch!")
    }
}

func TestStringEncoder(t *testing.T) {
    fixstr := "test"
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, fixstr)

    //Check fixstr
    if bytes.Compare(buf.Bytes(), []byte{ 0xa4, 't', 'e', 's', 't' }) != 0 {
        panic("Bytes mismatch!")
    }

    //Setup 31 > x < 256
    //0x21 to 0x7e
    sb := strings.Builder{}
    sb.Grow(240)
    generateChar(&sb, 240)
    encodeDebug(t, enc, &buf, sb.String())

    //Setup 255 > x < 65536
    sb = strings.Builder{}
    generateChar(&sb, 59999)
    encodeDebug(t, enc, &buf, sb.String())

    //Setup >65535
    sb = strings.Builder{}
    generateChar(&sb, 70321)
    encodeDebug(t, enc, &buf, sb.String())
}

// Test array
func TestArray(t *testing.T) {
    ints := []int{ -3, 16500, 1<<20, 0x0033ffaabbcceeff }
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, ints)
}

func TestMap(t *testing.T) {
    maps := map[string]int{ "test": 4, "gogo": 4 }
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, maps)
}

// Test struct by comparing to map
func TestStruct(t *testing.T) {
    maps := map[string]int{ "test": 2, "gogo": 5 }
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, maps)

    //Struct
    st := struct{ Test int `msgpack:"test"`
                  Gogo int `msgpack:"gogo"` } { maps["test"], maps["gogo"] }

    //Struct buffer
    stbuf := bytes.Buffer{}
    enc = NewEncoder(&stbuf)
    encodeDebug(t, enc, &stbuf, st)
    t.Logf("%.*s", stbuf.Len(), stbuf.Bytes())
}

// Test omitempty struct tag keyword
func TestOmitEmpty(t *testing.T) {
    mk := "audi"
    st := struct{ Omit *string   `msgpack:"omit,omitempty"`
                  Nil *string
                  Make *string }{ nil, nil, &mk }

    //Check omitted
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, &st)
    t.Logf("%.*s", buf.Len(), buf.Bytes())

    //Check not omitted
    buf.Reset()
    omit := "NO Omit!!"
    st.Omit = &omit
    enc = NewEncoder(&buf)
    encodeDebug(t, enc, &buf, &st)
    t.Logf("%.*s", buf.Len(), buf.Bytes())
}

// Test float
func TestFloat(t *testing.T) {
    f64 := float64(1.32342342341)
    f32 := float32(.992)

    //Encode floats
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, f64)
    encodeDebug(t, enc, &buf, f32)
    t.Logf("%.*s", buf.Len(), buf.Bytes())
}

// Test complicated struct
func TestComplicatedStruct(t *testing.T) {
    st := struct{ Make string
                  Model string
                  Year int
                  Properties map[string]string }{ "Audi", "A4", 2018, map[string]string{ "engine": "4-cylinder" } }

    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    encodeDebug(t, enc, &buf, &st)
    t.Logf("%.*s", buf.Len(), buf.Bytes())
}

// Test with the Marshal function
func TestMarshal(t *testing.T) {
    st := struct{ Make string
                  Model string
                  Year int
                  Properties map[string]string }{ "Audi", "A4", 2018, map[string]string{ "engine": "4-cylinder" } }

    //Marshal
    if buf, err := Marshal(st); err != nil {
        panic(fmt.Sprintf("%s", err.Error()))
    } else {
        t.Logf("%.*s", len(buf), buf)
    }
}

// General test function for decoding
func TestUnmarshal(t *testing.T) {
    var buf []byte
    num := uint64(392423)
    if b, err := Marshal(num); err != nil {
        panic(err)
    } else {
        buf = b
    }

    //Decode
    dnum := uint64(0)
    if err := Unmarshal(buf, &dnum); err != nil {
        panic(err)
    }

    //Check if match
    if dnum != num {
        panic(fmt.Sprintf("Decoded numbers are not the same as encoded! %v != %v", num, dnum))
    }

    //FixUint
    num = 12
    if b, err := Marshal(uint8(num)); err != nil {
        panic(err)
    } else {
        buf = b
    }

    //Decode
    dnum = 0
    if err := Unmarshal(buf, &dnum); err != nil {
        panic(err)
    }

    //Check if match
    if dnum != num {
        panic(fmt.Sprintf("Decoded numbers are not the same as encoded! %v != %v", num, dnum))
    }

    //Fixint
    inum := -5
    if b, err := Marshal(inum); err != nil {
        panic(err)
    } else {
        buf = b
    }

    //Decode
    var dinum int
    if err := Unmarshal(buf, &dinum); err != nil {
        panic(err)
    } else if inum != dinum {
        panic(fmt.Sprintf("Decoded numbers are not the same as encoded! %v != %v", inum, dinum))
    }
}
