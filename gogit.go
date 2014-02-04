package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type blob struct {
	sha1     string
	filename string
}

type tree struct {
	b     []*blob
	name  string
	child []*tree
}

type commit struct {
	sha1   string
	tree   *tree
	parent *commit
}

func (b *blob) checkout(prefix string) {
	if content, err := readSha1FileContent(b.sha1);err!=nil{
		log.Fatal("blob checkout error:", err)
	}else{
		body := getSha1FileContentBody(content)
		filename := prefix + "/" + b.filename
		log.Println("WriteFile:",filename)
		if err = ioutil.WriteFile(filename, body, 0644);err!=nil{
			log.Fatal("blob checkout error:", err)
		}
	}
}

func (t *tree) checkout(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println("Mkdir:",path)
		if err := os.Mkdir(path, 0777); err != nil {
			log.Fatal("mkdir error:", err)
			return
		}
	}
	for _, v := range t.b {
		v.checkout(path)	//BLOB checkout
	}
	for _, v := range t.child {
		v.checkout(path + "/" + v.name)		//TREE checkout
	}
}

func (c *commit) CheckOut() {
	if pwd, err := os.Getwd();err==nil{
		c.tree.checkout(pwd)
	}else{
		log.Fatal("commit checkout error:", err)
	}
}

func BuildCommit(sha1 string) (cmt *commit, hasParent bool) {
	cmt = new(commit)
	content, err := readSha1FileContent(sha1)
	if err != nil {
		log.Fatal("BuildCommit error:",err)
		return
	}
	cmt.sha1 = sha1
	start := 0
	for i := 0; i < len(content); i++ {
		v := content[i]
		if v == '\n' {
			line := string(content[start:i])
			fields := strings.Split(line, " ")
			switch fields[0] {
			case "commit": //TODO BuildTree
				cmt.tree = BuildTree(fields[2])
				break
			case "parent":
				return
				hasParent = true
				cmt.parent, _ = BuildCommit(fields[1])
				break
			}
			start = i + 1
			if content[i+1] == '\n' {
				break
			}
			i++
		}
	}
	return
}

func BuildTree(sha1 string) *tree {
	all, err := readSha1FileContent(sha1)
	if err != nil {
		log.Fatal("BuildTree error:", err)
		return nil
	}

	content := getSha1FileContentBody(all)
	start := 0
	tree := tree{}
	for i := 0; i < len(content); {
		if content[i] == 0 {
			line := content[start : i+21]
			_type := line[:6]
			id := line[i-start+1:]
			obj_sha1 := fmt.Sprintf("%x", id)
			switch string(_type[0:3]) {
			//BLOB
			case "100":
				name := string(line[7 : i-start])
				b := blob{sha1: obj_sha1, filename: name}
				tree.b = append(tree.b, &b)
				break
			//TREE
			case "400":
				name := string(line[6 : i-start])
				child := BuildTree(obj_sha1)
				child.name = name
				tree.child = append(tree.child, child)
				break
			}
			i += 21
			start = i
		} else {
			i++
		}
	}
	return &tree
}

func readSha1FileReader(sha1 string) (reader io.Reader, err error) {

	f, err := os.Open(getSha1FilePath(sha1))
	if err != nil{
		return
	}
	return zlib.NewReader(f)
}

func readSha1FileContent(sha1 string) (content []byte, err error) {

	if reader, err := readSha1FileReader(sha1);err == nil{
		buf := new(bytes.Buffer)
		buf.ReadFrom(reader)
		content = buf.Bytes()
	}
	return
}

func getSha1FileContentBody(content []byte) []byte {
	i := bytes.IndexByte(content, 0)
	return content[i+1:]
}

func getSha1FilePath(sha1 string) string {
	return ".git/objects/" + sha1[0:2] + "/" + sha1[2:]
}

func main() {
	master, err := ioutil.ReadFile(".git/refs/heads/master")
	if err != nil {
		log.Println(err)
	}
	commitTree, _ := BuildCommit(string(master[0 : len(master)-1]))
	commitTree.CheckOut()
}
