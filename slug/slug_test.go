package slug

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPack(t *testing.T) {
	dir, err := ioutil.TempDir("", "slug")
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "slug.tar.gz")

	meta, err := Pack("test-fixtures/archive-dir", out)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	raw, err := ioutil.ReadFile(out)
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	gzipR, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("err:%v", err)
	}

	tarR := tar.NewReader(gzipR)
	var (
		symFound bool
		fileList []string
		slugSize int64
	)
	for {
		hdr, err := tarR.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		fileList = append(fileList, hdr.Name)
		if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
			slugSize += hdr.Size
		}

		if hdr.Name == "sub/foo.txt" {
			if hdr.Typeflag != tar.TypeSymlink {
				t.Fatalf("expect symlink for file 'sub/foo.txt'")
			}
			if hdr.Linkname != "../foo.txt" {
				t.Fatalf("expect target of '../foo.txt', got %q", hdr.Linkname)
			}
			symFound = true
		}
	}

	// Make sure we saw and handled a symlink
	if !symFound {
		t.Fatal("expected to find symlink")
	}

	// Check the metadata
	expect := &SlugMeta{
		Files: fileList,
		Size:  slugSize,
	}
	if !reflect.DeepEqual(meta, expect) {
		t.Fatalf("\nexpect:\n%#v\n\nactual:\n%#v", expect, meta)
	}
}

func TestUnpack(t *testing.T) {
	dir, err := ioutil.TempDir("", "slug")
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "slug.tar.gz")

	// First create the slug file so we can try to unpack it.
	if _, err := Pack("test-fixtures/archive-dir", in); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Open the slug file for reading.
	fh, err := os.Open(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create a dir to unpack into.
	out, err := ioutil.TempDir(dir, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer os.RemoveAll(out)

	// Now try unpacking it.
	if err := Unpack(fh, out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify all the files
	verifyFile(t, filepath.Join(out, "foo.txt"), 0, "foo\n")
	verifyFile(t, filepath.Join(out, "bar.txt"), 0, "bar\n")
	verifyFile(t, filepath.Join(out, "sub", "zip.txt"), 0, "zip\n")
	verifyFile(t, filepath.Join(out, "sub", "foo.txt"), os.ModeSymlink, "../foo.txt")

	// Check that we can set permissions properly
	verifyPerms(t, filepath.Join(out, "foo.txt"), 0644)
	verifyPerms(t, filepath.Join(out, "bar.txt"), 0644)
	verifyPerms(t, filepath.Join(out, "sub", "zip.txt"), 0644)
	verifyPerms(t, filepath.Join(out, "sub", "foo.txt"), 0644)
	verifyPerms(t, filepath.Join(out, "exe"), 0755)
}

func TestUnpackDuplicateNoWritePerm(t *testing.T) {
	dir, err := ioutil.TempDir("", "slug")
	if err != nil {
		t.Fatalf("err:%v", err)
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "slug.tar.gz")

	// Create the output file
	wfh, err := os.Create(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Gzip compress all the output data
	gzipW := gzip.NewWriter(wfh)

	// Tar the file contents
	tarW := tar.NewWriter(gzipW)

	var hdr tar.Header

	data := "this is a\n"

	hdr.Name = "a"
	hdr.Mode = 0100000 | 0400
	hdr.Size = int64(len(data))

	tarW.WriteHeader(&hdr)
	tarW.Write([]byte(data))

	// write it twice
	tarW.WriteHeader(&hdr)
	tarW.Write([]byte(data))

	tarW.Close()
	gzipW.Close()
	wfh.Close()

	// Open the slug file for reading.
	fh, err := os.Open(in)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create a dir to unpack into.
	out, err := ioutil.TempDir(dir, "")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer os.RemoveAll(out)

	// Now try unpacking it.
	if err := Unpack(fh, out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify all the files
	verifyFile(t, filepath.Join(out, "a"), 0, "this is a\n")

	// Check that we can set permissions properly
	verifyPerms(t, filepath.Join(out, "a"), 0400)
}

func TestCheckFileMode(t *testing.T) {
	for _, tc := range []struct {
		desc string
		mode os.FileMode
		keep bool
		body bool
	}{
		{"includes regular files", 0, true, true},
		{"includes directories", os.ModeDir, true, false},
		{"includes symlinks", os.ModeSymlink, true, false},
		{"excludes unrecognized modes", os.ModeDevice, false, false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			keep, body := checkFileMode(tc.mode)
			if keep != tc.keep || body != tc.body {
				t.Fatalf("expect (%v, %v), got (%v, %v)",
					tc.keep, tc.body, keep, body)
			}
		})
	}
}

func verifyFile(t *testing.T, path string, mode os.FileMode, expect string) {
	fh, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()

	info, err := fh.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !((mode == 0 && info.Mode().IsRegular()) || info.Mode()&mode == 0) {
		t.Fatalf("wrong file mode for %q", path)
	}

	if mode == os.ModeSymlink {
		if target, _ := os.Readlink(path); target != expect {
			t.Fatalf("expect link target %q, got %q", expect, target)
		}
		return
	}

	raw := make([]byte, info.Size())
	if _, err := fh.Read(raw); err != nil {
		t.Fatal(err)
	}
	if result := string(raw); result != expect {
		t.Fatalf("bad content in file %q\n\nexpect:\n%#v\n\nactual:\n%#v",
			path, expect, result)
	}
}

func verifyPerms(t *testing.T, path string, expect os.FileMode) {
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != expect {
		t.Fatalf("expect perms %o, got %o", expect, perm)
	}
}
