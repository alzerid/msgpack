package msgpack

type Kind byte

// Signed integers
const (
    Int8 Kind = iota + 0xd0
    Int16
    Int32
    Int64
)

// Unsigned integers
const (
    Uint8 Kind = iota + 0xcc
    Uint16
    Uint32
    Uint64
)

// FixNums
const (
    FixInt Kind = 0xe0   // 0x0XXXXXXX (0 == control bit)
    FixUint Kind = 0x00  // 0x111YYYYY (111 == control bit)
)

// String interface for Kind type
func (k Kind) String() string {
    switch k {
        case Int64:
            return "Int64"
        case Int32:
            return "Int32"
        case Int16:
            return "Int16"
        case Int8:
            return "Int8"

        case Uint64:
            return "Uint64"
        case Uint32:
            return "Uint32"
        case Uint16:
            return "Uint16"
        case Uint8:
            return "Uint8"

        case FixInt:
           return "FixInt"
        case FixUint:
           return "FixUint"
    }

    return "unknown"
}
