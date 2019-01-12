package git

/*
#include <git2.h>
#include <git2/sys/openssl.h>
*/
import "C"
import (
	"bytes"
	"encoding/hex"
	"errors"
	"runtime"
	"strings"
	"unsafe"
)

type ErrorClass int

const (
	ErrClassNone       ErrorClass = C.GITERR_NONE
	ErrClassNoMemory   ErrorClass = C.GITERR_NOMEMORY
	ErrClassOs         ErrorClass = C.GITERR_OS
	ErrClassInvalid    ErrorClass = C.GITERR_INVALID
	ErrClassReference  ErrorClass = C.GITERR_REFERENCE
	ErrClassZlib       ErrorClass = C.GITERR_ZLIB
	ErrClassRepository ErrorClass = C.GITERR_REPOSITORY
	ErrClassConfig     ErrorClass = C.GITERR_CONFIG
	ErrClassRegex      ErrorClass = C.GITERR_REGEX
	ErrClassOdb        ErrorClass = C.GITERR_ODB
	ErrClassIndex      ErrorClass = C.GITERR_INDEX
	ErrClassObject     ErrorClass = C.GITERR_OBJECT
	ErrClassNet        ErrorClass = C.GITERR_NET
	ErrClassTag        ErrorClass = C.GITERR_TAG
	ErrClassTree       ErrorClass = C.GITERR_TREE
	ErrClassIndexer    ErrorClass = C.GITERR_INDEXER
	ErrClassSSL        ErrorClass = C.GITERR_SSL
	ErrClassSubmodule  ErrorClass = C.GITERR_SUBMODULE
	ErrClassThread     ErrorClass = C.GITERR_THREAD
	ErrClassStash      ErrorClass = C.GITERR_STASH
	ErrClassCheckout   ErrorClass = C.GITERR_CHECKOUT
	ErrClassFetchHead  ErrorClass = C.GITERR_FETCHHEAD
	ErrClassMerge      ErrorClass = C.GITERR_MERGE
	ErrClassSsh        ErrorClass = C.GITERR_SSH
	ErrClassFilter     ErrorClass = C.GITERR_FILTER
	ErrClassRevert     ErrorClass = C.GITERR_REVERT
	ErrClassCallback   ErrorClass = C.GITERR_CALLBACK
	ErrClassRebase     ErrorClass = C.GITERR_REBASE
)

type ErrorCode int

const (

	// No error
	ErrOk ErrorCode = C.GIT_OK

	// Generic error
	ErrGeneric ErrorCode = C.GIT_ERROR
	// Requested object could not be found
	ErrNotFound ErrorCode = C.GIT_ENOTFOUND
	// Object exists preventing operation
	ErrExists ErrorCode = C.GIT_EEXISTS
	// More than one object matches
	ErrAmbigious ErrorCode = C.GIT_EAMBIGUOUS
	// Output buffer too short to hold data
	ErrBuffs ErrorCode = C.GIT_EBUFS

	// GIT_EUSER is a special error that is never generated by libgit2
	// code.  You can return it from a callback (e.g to stop an iteration)
	// to know that it was generated by the callback and not by libgit2.
	ErrUser ErrorCode = C.GIT_EUSER

	// Operation not allowed on bare repository
	ErrBareRepo ErrorCode = C.GIT_EBAREREPO
	// HEAD refers to branch with no commits
	ErrUnbornBranch ErrorCode = C.GIT_EUNBORNBRANCH
	// Merge in progress prevented operation
	ErrUnmerged ErrorCode = C.GIT_EUNMERGED
	// Reference was not fast-forwardable
	ErrNonFastForward ErrorCode = C.GIT_ENONFASTFORWARD
	// Name/ref spec was not in a valid format
	ErrInvalidSpec ErrorCode = C.GIT_EINVALIDSPEC
	// Checkout conflicts prevented operation
	ErrConflict ErrorCode = C.GIT_ECONFLICT
	// Lock file prevented operation
	ErrLocked ErrorCode = C.GIT_ELOCKED
	// Reference value does not match expected
	ErrModified ErrorCode = C.GIT_EMODIFIED
	// Authentication failed
	ErrAuth ErrorCode = C.GIT_EAUTH
	// Server certificate is invalid
	ErrCertificate ErrorCode = C.GIT_ECERTIFICATE
	// Patch/merge has already been applied
	ErrApplied ErrorCode = C.GIT_EAPPLIED
	// The requested peel operation is not possible
	ErrPeel ErrorCode = C.GIT_EPEEL
	// Unexpected EOF
	ErrEOF ErrorCode = C.GIT_EEOF
	// Uncommitted changes in index prevented operation
	ErrUncommitted ErrorCode = C.GIT_EUNCOMMITTED
	// The operation is not valid for a directory
	ErrDirectory ErrorCode = C.GIT_EDIRECTORY
	// A merge conflict exists and cannot continue
	ErrMergeConflict ErrorCode = C.GIT_EMERGECONFLICT

	// Internal only
	ErrPassthrough ErrorCode = C.GIT_PASSTHROUGH
	// Signals end of iteration with iterator
	ErrIterOver ErrorCode = C.GIT_ITEROVER
)

var (
	ErrInvalid = errors.New("Invalid state for operation")
)

var pointerHandles *HandleList
var remotePointers *remotePointerList

func init() {
	pointerHandles = NewHandleList()
	remotePointers = newRemotePointerList()

	C.git_libgit2_init()

	// Due to the multithreaded nature of Go and its interaction with
	// calling C functions, we cannot work with a library that was not built
	// with multi-threading support. The most likely outcome is a segfault
	// or panic at an incomprehensible time, so let's make it easy by
	// panicking right here.
	if Features()&FeatureThreads == 0 {
		panic("libgit2 was not built with threading support")
	}

	if err := registerManagedHttp(); err != nil {
		panic(err)
	}

	if err := registerManagedHttps(); err != nil {
		panic(err)
	}
}

// Oid represents the id for a Git object.
type Oid [20]byte

func newOidFromC(coid *C.git_oid) *Oid {
	if coid == nil {
		return nil
	}

	oid := new(Oid)
	copy(oid[0:20], C.GoBytes(unsafe.Pointer(coid), 20))
	return oid
}

func NewOidFromBytes(b []byte) *Oid {
	oid := new(Oid)
	copy(oid[0:20], b[0:20])
	return oid
}

func (oid *Oid) toC() *C.git_oid {
	return (*C.git_oid)(unsafe.Pointer(oid))
}

func NewOid(s string) (*Oid, error) {
	if len(s) > C.GIT_OID_HEXSZ {
		return nil, errors.New("string is too long for oid")
	}

	o := new(Oid)

	slice, error := hex.DecodeString(s)
	if error != nil {
		return nil, error
	}

	if len(slice) != 20 {
		return nil, &GitError{"Invalid Oid", ErrClassNone, ErrGeneric}
	}

	copy(o[:], slice[:20])
	return o, nil
}

func (oid *Oid) String() string {
	return hex.EncodeToString(oid[:])
}

func (oid *Oid) Cmp(oid2 *Oid) int {
	return bytes.Compare(oid[:], oid2[:])
}

func (oid *Oid) Copy() *Oid {
	ret := *oid
	return &ret
}

func (oid *Oid) Equal(oid2 *Oid) bool {
	return *oid == *oid2
}

func (oid *Oid) IsZero() bool {
	return *oid == Oid{}
}

func (oid *Oid) NCmp(oid2 *Oid, n uint) int {
	return bytes.Compare(oid[:n], oid2[:n])
}

func ShortenOids(ids []*Oid, minlen int) (int, error) {
	shorten := C.git_oid_shorten_new(C.size_t(minlen))
	if shorten == nil {
		panic("Out of memory")
	}
	defer C.git_oid_shorten_free(shorten)

	var ret C.int

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for _, id := range ids {
		buf := make([]byte, 41)
		C.git_oid_fmt((*C.char)(unsafe.Pointer(&buf[0])), id.toC())
		buf[40] = 0
		ret = C.git_oid_shorten_add(shorten, (*C.char)(unsafe.Pointer(&buf[0])))
		if ret < 0 {
			return int(ret), MakeGitError(ret)
		}
	}
	runtime.KeepAlive(ids)
	return int(ret), nil
}

type GitError struct {
	Message string
	Class   ErrorClass
	Code    ErrorCode
}

func (e GitError) Error() string {
	return e.Message
}

func IsErrorClass(err error, c ErrorClass) bool {

	if err == nil {
		return false
	}
	if gitError, ok := err.(*GitError); ok {
		return gitError.Class == c
	}
	return false
}

func IsErrorCode(err error, c ErrorCode) bool {
	if err == nil {
		return false
	}
	if gitError, ok := err.(*GitError); ok {
		return gitError.Code == c
	}
	return false
}

func MakeGitError(errorCode C.int) error {
	var errMessage string
	var errClass ErrorClass
	if errorCode != C.GIT_ITEROVER {
		err := C.giterr_last()
		if err != nil {
			errMessage = C.GoString(err.message)
			errClass = ErrorClass(err.klass)
		} else {
			errClass = ErrClassInvalid
		}
	}
	return &GitError{errMessage, errClass, ErrorCode(errorCode)}
}

func MakeGitError2(err int) error {
	return MakeGitError(C.int(err))
}

func setLibgit2Error(errorClass ErrorClass, err error) C.int {
	cstr := C.CString(err.Error())
	defer C.free(unsafe.Pointer(cstr))
	C.giterr_set_str(C.int(errorClass), cstr)

	if gitErr, ok := err.(*GitError); ok {
		return C.int(gitErr.Code)
	}

	return -1
}

func cbool(b bool) C.int {
	if b {
		return C.int(1)
	}
	return C.int(0)
}

func ucbool(b bool) C.uint {
	if b {
		return C.uint(1)
	}
	return C.uint(0)
}

func Discover(start string, across_fs bool, ceiling_dirs []string) (string, error) {
	ceildirs := C.CString(strings.Join(ceiling_dirs, string(C.GIT_PATH_LIST_SEPARATOR)))
	defer C.free(unsafe.Pointer(ceildirs))

	cstart := C.CString(start)
	defer C.free(unsafe.Pointer(cstart))

	var buf C.git_buf
	defer C.git_buf_dispose(&buf)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ret := C.git_repository_discover(&buf, cstart, cbool(across_fs), ceildirs)
	if ret < 0 {
		return "", MakeGitError(ret)
	}

	return C.GoString(buf.ptr), nil
}
