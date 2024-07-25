package smb

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	iofs "io/fs"

	"github.com/hirochachacha/go-smb2"
	"github.com/kosmosec/mykmyk/internal/credsmanager"
)

func scan(host string, args []string, creds credsmanager.Credentials) (string, error) {
	if _, err := os.Stat(host); os.IsNotExist(err) {
		os.Mkdir(host, 0775)
	}

	output, err := checkUserSession(host, creds)
	if err != nil {
		return "", nil
	}
	return output, nil

}

func checkUserSession(hostname string, creds credsmanager.Credentials) (string, error) {
	fmt.Printf("[+] SMB scanning for %s started\n", hostname)
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", hostname, 445))
	if err != nil {
		return "", err
	}

	defer conn.Close()

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     creds.User,
			Password: creds.Password,
			Domain:   creds.Domain,
		},
	}

	s, err := d.Dial(conn)
	if err != nil {
		return "", err
	}
	defer s.Logoff()

	names, err := s.ListSharenames()
	if err != nil {
		return "", err
	}
	var output string
	output += fmt.Sprintf("List SMB shares for %s\n", hostname)
	for _, name := range names {
		perm := checkPermissions(s, name)
		output += fmt.Sprintf("\t|%-30s|%-18s|\n", name, perm)
		if name == "ldlogon" {
			listFolders(s, name)
		}
	}

	return output, nil
}

func listFolders(s *smb2.Session, name string) {
	fs, err := s.Mount(name)
	if err != nil {
		return
	}
	defer fs.Umount()

	// matches, err := iofs.Glob(fs.DirFS("."), "*")
	// if err != nil {
	// 	return
	// }
	// for _, match := range matches {
	// 	fmt.Println(match)
	// }
	files := []string{}

	err = iofs.WalkDir(fs.DirFS("."), ".", func(path string, d iofs.DirEntry, err error) error {
		if d.IsDir() {

			absolutefilepath, err := filepath.Abs(path)
			//absolutefilepath := path
			if err != nil {
				log.Fatal(err)
			}
			files = append(files, absolutefilepath)

		}
		return nil
	})
	var output string
	for _, dd := range files {
		perm := checkPermissions2(s, fs, dd)
		output += fmt.Sprintf("\t|%-60s|%-18s|\n", dd, perm)
		fmt.Println(output)
	}
	if err != nil {
		return
	}
}

func checkPermissions2(s *smb2.Session, fs *smb2.Share, dirName string) string {
	var canWrite bool
	var canRead bool
	_, err := fs.ReadDir(".")
	if err == nil {
		canRead = true
	}
	// for _, ii := range sharedFiles {
	// 	fmt.Println(ii.Mode().Perm().String())
	// }
	f, err := fs.Create(dirName + "/aaaPentestaaa.txt")
	//err = fs.Mkdir(dirName+"/aaaPentestaaa", 0700)
	if err == nil {
		defer f.Close()
		canWrite = true
		err = fs.Remove(dirName + "/aaaPentestaaa")
		if err != nil {
			fmt.Printf("unable to remove test file at %s, please remove manually. %s\n", dirName, err)
		}
	}

	if canRead && canWrite {
		return "READ WRITE"
	} else if canRead {
		return "READ"
	} else if canWrite {
		return "WRITE"
	} else {
		return "NO WRITE AND READ"
	}
}

func checkPermissions(s *smb2.Session, shareName string) string {
	var canWrite bool
	var canRead bool

	fs, err := s.Mount(shareName)
	if err != nil {
		return "NO_ACCESS"
	}

	defer fs.Umount()

	_, err = fs.ReadDir(".")
	if err == nil {
		canRead = true
	}
	// for _, ii := range sharedFiles {
	// 	fmt.Println(ii.Mode().Perm().String())
	// }

	err = fs.Mkdir("aaaPentestaaa", 0700)
	if err == nil {
		canWrite = true
		err = fs.Remove("aaaPentestaaa")
		if err != nil {
			fmt.Printf("unable to remove test file at %s, please remove manually. %s\n", shareName, err)
		}
	}

	if canRead && canWrite {
		return "READ WRITE"
	} else if canRead {
		return "READ"
	} else if canWrite {
		return "WRITE"
	} else {
		return "NO WRITE AND READ"
	}

}
