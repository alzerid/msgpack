package msgpack
// https://github.com/msgpack/msgpack/blob/master/spec.md#int-format-family
import (
    "reflect"
    "strings"
    "unsafe"
    "bytes"
    "log"
    "fmt"
    "io"
)

type Marshaler interface {
    MsgPackMarshaler() ([]byte, error)
}

type Encoder struct {
    wtr io.Writer
}

// Convineince function to write out a byte onto a writer
func writeByte(wtr io.Writer, b byte) error {
    if bwtr, ok := wtr.(io.ByteWriter); ok {
        return bwtr.WriteByte(b)
    }

    _, err := wtr.Write([]byte{ b })
    return err
}

// Function reverses a byte array (BE to LE and LE to BE convesion)
func reverseByte(b []byte) {
    blen := len(b)
    for i:=0; i<blen/2; i++ {
        b[i], b[blen-i-1] = b[blen-i-1], b[i]
    }
}

var errFixNumOutOfBounds = fmt.Errorf("Numerical value out of bounds for fixint")

// Function takes in an integer and translates the number to a fixnum format
// Fixnum format consists of an 8bit (1byte) integer with positive numbers:
// 0XXXXXXX where MSB is the control bit set to 0 and the next 7bits are the
// positive number
func encodeFixNumInt(wtr io.Writer, val int8) error {
    // Integer signed 
    // For positive integers we can store 0XXXXXXX
    // where 0 is postive int indicator and XXXXXXX is
    // actual number
    if val <= 0x7F && val >= 0 {
        nval := byte(0x00 | (uint8(val) & 0x7F))
        if bwtr, ok := wtr.(io.ByteWriter); ok {
            bwtr.WriteByte(nval)
        } else {
            wtr.Write([]byte{ nval })
        }

        return nil

    // Negative number representations are 111YYYYY
    // Control bits are 111 and YYYYY is the value of
    // the negative integer
    // We convert the negative integer two's compliment
    // to postive to ensure it fits in 5 bits (YYYYY)
    } else if val < 0 && ((uint8(val) ^ 0xFF)+1) < 0x1F {
        // Ones compliment the number
        //      111b       YYYYY (5 bit)
        nval := 0xE0 | byte((uint8(val) ^ 0xFF)+1)
        if bwtr, ok := wtr.(io.ByteWriter); ok {
            bwtr.WriteByte(nval)
        } else {
            wtr.Write([]byte{ nval })
        }

        return nil
    }

    return errFixNumOutOfBounds
}

// Function encodes the integer based on size
func EncodeInt(wtr io.Writer, val int64, bsize int) error {
    bsize /= 8

    //Check if we do FixNum int encoding
    if (val <= 0x7f && val >= 0) || (val < 0 && ((val ^ 0xff)+1) < 0x1f) {
        return encodeFixNumInt(wtr, int8(val))
    }

    // All representations are in big endian format. Msgpack
    // representation is sizeof(val) + 1 where the first byte
    // is the control byte that specifies size of the value.
    // 0xd0 - 1 byte value
    // 0xd1 - 2 byte value
    // 0xd2 - 4 byte value
    // 0xd3 - 8 byte value
    var ctlbyte byte
    var bval []byte
    if bsize == 1 {
        ctlbyte = 0xd0
        bval = (*[1]byte)(unsafe.Pointer(&val))[:]
    } else if bsize == 2 {
        ctlbyte = 0xd1
        bval = (*[2]byte)(unsafe.Pointer(&val))[:]
    } else if bsize == 4 {
        ctlbyte = 0xd2
        bval = (*[4]byte)(unsafe.Pointer(&val))[:]
    } else if bsize == 8 {
        ctlbyte = 0xd3
        bval = (*[8]byte)(unsafe.Pointer(&val))[:]
    }

    //Write byte
    if err := writeByte(wtr, ctlbyte); err != nil {
        return err
    }

    //TODO: Check endianess of machine, we are assuming we are running on a little endian arch
    reverseByte(bval)
    if _, err := wtr.Write(bval); err != nil {
        return err
    }

    return nil
}

// Function encodes the unsigned integer based on size
func EncodeUint(wtr io.Writer, val uint64, bsize int) error {
    bsize /= 8

    //Check if we do FixNum int encoding
    if val <= 0x7f && val >= 0 {
        return encodeFixNumInt(wtr, int8(val))
    }

    // All representations are in big endian format. Msgpack
    // representation is sizeof(val) + 1 where the first byte
    // is the control byte that specifies size of the value.
    // 0xcc - 1 byte value
    // 0xcd - 2 byte value
    // 0xce - 4 byte value
    // 0xcf - 8 byte value
    var ctlbyte byte
    var bval []byte
    if bsize == 1 {
        ctlbyte = 0xcc
        bval = (*[1]byte)(unsafe.Pointer(&val))[:]
    } else if bsize == 2 {
        ctlbyte = 0xcd
        bval = (*[2]byte)(unsafe.Pointer(&val))[:]
    } else if bsize == 4 {
        ctlbyte = 0xce
        bval = (*[4]byte)(unsafe.Pointer(&val))[:]
    } else if bsize == 8 {
        ctlbyte = 0xcf
        bval = (*[8]byte)(unsafe.Pointer(&val))[:]
    }

    //Write byte
    if err := writeByte(wtr, ctlbyte); err != nil {
        return err
    }

    //TODO: Check endianess of machine, we are assuming we are running on a little endian arch
    reverseByte(bval)
    if _, err := wtr.Write(bval); err != nil {
        return err
    }

    return nil
}

// Function encodes a boolean into msgpack
func EncodeBool(wtr io.Writer, val bool) error {
    var bval byte
    if val {
        bval = 0xc3
    } else {
        bval = 0xc2
    }

    //Write out
    if bwtr, ok := wtr.(io.ByteWriter); ok {
        bwtr.WriteByte(bval)
    } else {
        wtr.Write([]byte{ bval })
    }

    return nil
}

// Function encodes the string into msgpack format
// The first few bytes are a control byte + length
// definition of the string. Strings are not NULL
// terminated.
// | 101XXXXX | data - [fixstr] For data that is <=31 bytes long
// | 0xd9 | YYYYYYYY | data - [str8] For data with lengths that are 255 or less
// | 0xda | ZZZZZZZZ | ZZZZZZZZ | data - [str16] For data with lengths that are 65535 or less
// | 0xdb | AAAAAAAA * 4 | data - [str32] For data with lengths that are 4294967295 or less
func EncodeString(wtr io.Writer, s string) error {
    //Write control byte
    l := len(s)
    switch {
        case l<=31:
            var err error
            lval := byte(0xa0 | l)
            if bwtr, ok := wtr.(io.ByteWriter); ok {
                err = bwtr.WriteByte(lval)
            } else {
                _, err = wtr.Write([]byte{ lval })
            }

            //Check if we errored out
            if err != nil {
                return err
            }

        case l <= 255:
            //Control byte
            if err := writeByte(wtr, 0xd9); err != nil {
                return err
            }

            //Length of string
            if err := writeByte(wtr, byte(l)); err != nil {
                return err
            }

        case l <= 65535:
            //Control byte
            if err := writeByte(wtr, 0xda); err != nil {
                return err
            }

            //Length of string
            var bval []byte
            tlen := uint16(l)
            bval = (*[2]byte)(unsafe.Pointer(&tlen))[:]
            reverseByte(bval)
            if _, err := wtr.Write(bval); err != nil {
                return err
            }

        case l <= 4294967295:
            //Control byte
            if err := writeByte(wtr, 0xdb); err != nil {
                return err
            }

            //Length of string 4 bytes
            var bval []byte
            tlen := uint32(l)
            bval = (*[4]byte)(unsafe.Pointer(&tlen))[:]
            reverseByte(bval)
            if _, err := wtr.Write(bval); err != nil {
                return err
            }

        default:
            panic("String larger than 4 gigs!")
    }

    //Write out the string
    _, err := io.WriteString(wtr, s)
    return err
}

// Function encodes a byte array to a binary msgpack type
// bin format has three control bytes we can use:
// | 0xc4 | XXXXXXXX | data - 255 length binary data
// | 0xc5 | XXXXXXXX * 2 | data - 65535 length binary data
// | 0xc6 | XXXXXXXX * 4 | data - 4294967295 length binary data
func EncodeBin(wtr io.Writer, b []byte) error {
    l := len(b)
    switch {
        case l<=255:
            //Control byte
            if err := writeByte(wtr, 0xc4); err != nil {
                return err
            }

            //Lenght of binary data 1 byte
            if err := writeByte(wtr, byte(l)); err != nil {
                return err
            }

        case l <= 65535:
            //Control Byte
            if err := writeByte(wtr, 0xc5); err != nil {
                return err
            }

            //Length of binary data 2 bytes
            var bval []byte
            tlen := uint16(l)
            bval = (*[2]byte)(unsafe.Pointer(&tlen))[:]
            reverseByte(bval)
            if _, err := wtr.Write(bval); err != nil {
                return err
            }

        case l <= 4294967295:
            //Control byte
            if err := writeByte(wtr, 0xc6); err != nil {
                return err
            }

            //Length of string 4 bytes
            var bval []byte
            tlen := uint32(l)
            bval = (*[4]byte)(unsafe.Pointer(&tlen))[:]
            reverseByte(bval)
            if _, err := wtr.Write(bval); err != nil {
                return err
            }

        default:
            panic("Binary larger than 4 gigs!")
    }

    //Write the binary data
    _, err := wtr.Write(b)
    return err
}

// Encode float 64
// Float64 is a 9 byte binary (1 byte control + 8 byte float)
// The data portion must be big endian format
// | 0xca | XXXXXXXX * 8 |
func EncodeFloat64(wtr io.Writer, f float64) error {
    //Write control byte
    if err := writeByte(wtr, 0xca); err != nil {
        return err
    }

    //Convert float to byte order and reverse (LE to BE)
    b := (*[unsafe.Sizeof(f)]byte)(unsafe.Pointer(&f))[:]
    reverseByte(b)
    if _, err := wtr.Write(b); err != nil {
        return err
    }

    return nil
}

// Encode float 32
// Float64 is a 5 byte binary (1 byte control + 4 byte float)
// The data portion must be big endian format
// | 0xcb | XXXXXXXX * 4 |
func EncodeFloat32(wtr io.Writer, f float32) error {
    //Write control byte
    if err := writeByte(wtr, 0xca); err != nil {
        return err
    }

    //Convert float to byte order and reverse (LE to BE)
    b := (*[unsafe.Sizeof(f)]byte)(unsafe.Pointer(&f))[:]
    reverseByte(b)
    if _, err := wtr.Write(b); err != nil {
        return err
    }

    return nil
}
// Function encodes the nil value
func EncodeNil(wtr io.Writer) error {
    return writeByte(wtr, 0xc0)
}

/* Start Encoder **/

// Function creates a new encoder
func NewEncoder(w io.Writer) *Encoder {
    return &Encoder{ wtr: w }
}

// Function encodes an array into the writer
// msgpack defines three array encoding types
// | 1001XXXX | data - [fixarray] up to 15 elements
// | 0xdc | YYYYYYYY * 2 | data - [array16] stores up to 65535 elements
// | 0xdd | ZZZZZZZZ * 4 | data - [array32] stores up to 4294967295 elements
func (e *Encoder) encodeArray(typ reflect.Type, val reflect.Value) error {
    l := val.Len()
    switch {
        case l <= 15:
            //Control byte + len
            if err := writeByte(e.wtr, byte(0x90 | l)); err != nil {
                return err
            }

        case l <= 65535:
            //Control byte
            if err := writeByte(e.wtr, 0xdc); err != nil {
                return err
            }

            //Big Endian the length
            var bval []byte
            tlen := uint16(l)
            bval = (*[2]byte)(unsafe.Pointer(&tlen))[:]
            reverseByte(bval)
            if _, err := e.wtr.Write(bval); err != nil {
                return err
            }

        case l <= 4294967295:
            //Control byte
            if err := writeByte(e.wtr, 0xdd); err != nil {
                return err
            }

            //Big Endian the length
            var bval []byte
            tlen := uint32(l)
            bval = (*[4]byte)(unsafe.Pointer(&tlen))[:]
            reverseByte(bval)
            if _, err := e.wtr.Write(bval); err != nil {
                return err
            }

        default:
            panic("Array size larger than 4294967295!")
    }

    //Actual data
    for i:=0; i<l; i++ {
        ed := val.Index(i).Interface()
        if err := e.Encode(ed); err != nil {
            return err
        }
    }

    return nil
}

// Function encodes the header for a Map (hash)
func (e *Encoder) encodeMapHeader(l int) error {
    switch {
        case l <= 15:
            //Control byte + len
            if err := writeByte(e.wtr, byte(0x80 | l)); err != nil {
                return err
            }

        case l <= 65535:
            //Control byte
            if err := writeByte(e.wtr, 0xde); err != nil {
                return err
            }

            //Big Endian the length
            var bval []byte
            tlen := uint16(l)
            bval = (*[2]byte)(unsafe.Pointer(&tlen))[:]
            reverseByte(bval)
            if _, err := e.wtr.Write(bval); err != nil {
                return err
            }

        case l <= 4294967295:
            //Control byte
            if err := writeByte(e.wtr, 0xdf); err != nil {
                return err
            }

            //Big Endian the length
            var bval []byte
            tlen := uint32(l)
            bval = (*[4]byte)(unsafe.Pointer(&tlen))[:]
            reverseByte(bval)
            if _, err := e.wtr.Write(bval); err != nil {
                return err
            }

        default:
            panic("Array size larger than 4294967295!")
    }

    return nil
}

// Function encodes the map entity
func (e *Encoder) encodeMapEntity(key interface{}, val interface{}) error {
    //Encode key
    if err := e.Encode(key); err != nil {
        return err
    }

    //Encode value
    if err := e.Encode(val); err != nil {
        return err
    }

    return nil
}

// Function encodes a map into the writer
// msgpack defines three map encoding types
// | 1000XXXX | data - [fixmap] up to 15 elements
// | 0xde | YYYYYYYY * 2 | data - [map16] up to 65535 elements
// | 0xdf | YYYYYYYY * 4 | data - [map32] up to 4294967295 elements
func (e *Encoder) encodeMap(typ reflect.Type, val reflect.Value) error {
    l := val.Len()
    if err := e.encodeMapHeader(l); err != nil {
        return err
    }

    //Actual data
    keys := val.MapKeys()
    for i:=0; i<len(keys); i++ {
        if err := e.encodeMapEntity(keys[i].Interface(), val.MapIndex(keys[i]).Interface()); err != nil {
            return err
        }
    }

    return nil
}

// Function encodes a structs public entities.
// msgpack uses the "msgpack" tag for renaming
// of keys
// Structs are just maps:
// | 1000XXXX | data - [fixmap] up to 15 elements
// | 0xde | YYYYYYYY * 2 | data - [map16] up to 65535 elements
// | 0xdf | YYYYYYYY * 4 | data - [map32] up to 4294967295 elements
func (e *Encoder) encodeStruct(t reflect.Type, v reflect.Value) error {
    slen := v.NumField()
    if err := e.encodeMapHeader(slen); err != nil {
        return err
    }

    //Go through the struct
    for i:=0; i<slen; i++ {
        var tname *string
        stval := t.Field(i)

        //Get any msgpack tags
        if tval, ok := stval.Tag.Lookup("msgpack"); ok {
            if stmp, omit := parseMsgPackTag(tval); omit && stmp == "" {
                continue
            } else if stmp != "" {
                tname = &stmp
            }
        }

        //Struct type name
        if tname == nil {
            tname = &stval.Name
        }

        //Field value
        if err := e.encodeMapEntity(*tname, v.Field(i).Interface()); err != nil {
            return err
        }
    }

    return nil
}

// Function parses out the struct tag contents.
// First return is the value name of the tag.
// Second return is if we omit if empty (omitempty)
func parseMsgPackTag(t string) (fieldname string, omit bool) {
    sp := strings.Split(t, ",")
    if len(sp) >= 1 {
        fieldname = strings.TrimSpace(sp[0])
    }

    if len(sp) >= 2 && strings.TrimSpace(sp[1]) == "omitempty" {
        omit = true
    }

    return
}

// Function encodes the interface
func (e *Encoder) Encode(v interface{}) error {
    //Check for base type encoding
    switch val := v.(type) {

        //Integer
        case int64:
            return EncodeInt(e.wtr, val, 64)
        case int32:
            return EncodeInt(e.wtr, int64(val), 32)
        case int16:
            return EncodeInt(e.wtr, int64(val), 16)
        case int8:
            return EncodeInt(e.wtr, int64(val), 8)
        case int:
            return EncodeInt(e.wtr, int64(val), int(reflect.TypeOf(v).Size())*8)

        //Unsigned integer
        case uint64:
            return EncodeUint(e.wtr, val, 64)
        case uint32:
            return EncodeUint(e.wtr, uint64(val), 32)
        case uint16:
            return EncodeUint(e.wtr, uint64(val), 16)
        case uint8:
            return EncodeUint(e.wtr, uint64(val), 8)
        case uint:
            return EncodeUint(e.wtr, uint64(val), int(reflect.TypeOf(v).Size())*8)

        //Float
        case float64:
            return EncodeFloat64(e.wtr, float64(val))

        case float32:
            return EncodeFloat32(e.wtr, float32(val))

        //Boolean
        case bool:
            return EncodeBool(e.wtr, val)

        //String
        case string:
            return EncodeString(e.wtr, val)

        //Byte (binary)
        case []byte:
            return EncodeBin(e.wtr, val)

        //Nil case
        case nil:
            return EncodeNil(e.wtr)
    }

    //More complicated type discovery
    typ := reflect.TypeOf(v)
    switch typ.Kind() {
        //Array/slice type
        case reflect.Slice:
            fallthrough
        case reflect.Array:
            return e.encodeArray(typ, reflect.ValueOf(v))

        case reflect.Map:
            return e.encodeMap(typ, reflect.ValueOf(v))

        case reflect.Struct:
            return e.encodeStruct(typ, reflect.ValueOf(v))

        case reflect.Ptr:
            vptr := reflect.ValueOf(v)
            if vptr.IsNil() {
                return e.Encode(nil)
            }

            return e.Encode(reflect.Indirect(vptr).Interface())
    }

    //TODO: Encode ext
    log.Panicf("Unhandled type %T", v)
    return nil
}

// Marshal function
func Marshal(v interface{}) ([]byte, error) {
    buf := bytes.Buffer{}
    enc := NewEncoder(&buf)
    if err := enc.Encode(v); err != nil {
        return nil, err
    }

    return buf.Bytes(), nil
}
