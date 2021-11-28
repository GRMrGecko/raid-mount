package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// RaidMount: Mount point details.
type RaidMount struct {
	Source    string
	Target    string
	FSType    string
	Flags     string
	CryptName string
	Encrypted bool
}

// App: Global application structure.
type App struct {
	flags  *Flags
	config Config
}

var app *App

// isMounted: Checks the linux mounts for a target mountpoint to see if it is mounted.
func isMounted(target string) bool {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		args := strings.Fields(scanner.Text())
		if len(args) < 3 {
			continue
		}
		if args[1] == target {
			return true
		}
	}
	return false
}

// main: Starting application function.
func main() {
	// Only allow running as root.
	if os.Getuid() != 0 {
		fmt.Println("You must call this program as root.")
		os.Exit(1)
	}

	// Read configurations.
	app = new(App)
	app.flags = new(Flags)
	app.flags.Init()
	app.ReadConfig()

	// The raid table is how we know what to mount, and it must exist to start.
	if _, err := os.Stat(app.config.RaidTablePath); err != nil {
		log.Fatalln("Raid table does not exist.")
	}

	var raidMounts []RaidMount
	hasEncryptedDrives := false // If there are encrypted drives, we require a password to decrypt them.

	// Open the raid mountpoint table file.
	raidTab, err := os.Open(app.config.RaidTablePath)
	if err != nil {
		log.Fatalln("Unable to open raid table:", err)
	}

	// Prepare scanners and regular expressions for parsing raid table.
	scanner := bufio.NewScanner(raidTab)
	comment := regexp.MustCompile(`#.*`)
	uuidMatch := regexp.MustCompile(`^UUID=["]*([0-9a-f-]+)["]*$`)
	partuuidMatch := regexp.MustCompile(`^PARTUUID=["]*([0-9a-f-]+)["]*$`)

	// Each line item, parse the mountpoint.
	for scanner.Scan() {
		// Read line, and clean up comments/parse fields.
		line := scanner.Text()
		line = comment.ReplaceAllString(line, "")
		args := strings.Fields(line)

		// If line contains no fields, we can ignore it.
		if len(args) == 0 {
			continue
		}

		// If line is not 5 fields, some formatting is wrong in the table. We will just log/ignore this line.
		if len(args) != 5 {
			log.Println("Line does not have correct number of arguments:", line)
			continue
		}

		// Put fields into mountpoint structure.
		mount := RaidMount{
			Source:    strings.ReplaceAll(args[0], "\\040", " "),
			Target:    strings.ReplaceAll(args[1], "\\040", " "),
			FSType:    args[2],
			Flags:     args[3],
			CryptName: args[4],
			Encrypted: false,
		}

		// If the CryptName field is not none, then it is an encrypted drive. We must set the variables for logic below to easily determine if it has encryption.
		if mount.CryptName != "none" {
			mount.Encrypted = true
			hasEncryptedDrives = true
		}

		// If the source drive is a UUID or PARTUUID, expand to device name.
		if uuidMatch.MatchString(mount.Source) {
			uuid := uuidMatch.FindStringSubmatch(mount.Source)
			mount.Source = "/dev/disk/by-uuid/" + uuid[1]
		} else if partuuidMatch.MatchString(mount.Source) {
			uuid := partuuidMatch.FindStringSubmatch(mount.Source)
			mount.Source = "/dev/disk/by-partuuid/" + uuid[1]
		}

		raidMounts = append(raidMounts, mount)
	}
	raidTab.Close()

	// If the encryption key was passed as a flag, override the configuration file.
	if app.flags.EncryptionKey != "" {
		app.config.EncryptionKey = app.flags.EncryptionKey
	}

	// If the encryption key file is set, we need to verify it actually exists.
	if app.config.EncryptionKey != "" {
		if _, err := os.Stat(app.config.EncryptionKey); err != nil {
			log.Fatalln("Encryption key specified does not exist.")
		}
	}

	// If the encryption password was not provided and an encryption key not provided and there is a mountpoint that is encrypted,
	//  request the password from the user.
	encryptionPassword := app.flags.EncryptionPassword
	if encryptionPassword == "" && app.config.EncryptionKey == "" && hasEncryptedDrives {
		fmt.Print("Please enter the encryption password: ")

		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalln("Unable to read password:", err)
		}
		fmt.Println()

		encryptionPassword = string(bytePassword)
	}

	// With each mountpoint, decrypt and mount.
	for _, mount := range raidMounts {
		// If encrypted, decrypt the drive.
		if mount.Encrypted {
			// Check the device path to see if the encrypted drive is already decrypted.
			dmPath := "/dev/mapper/" + mount.CryptName
			if _, err := os.Stat(dmPath); err == nil {
				fmt.Println("Already decrypted:", mount.CryptName)
				continue
			}

			// Decrypt the drive.
			args := []string{
				"open",
				mount.Source,
				mount.CryptName,
			}

			// If encryption key file was provided, add argument.
			if app.config.EncryptionKey != "" {
				args = append(args, "--key-file="+app.config.EncryptionKey)
			}

			fmt.Println("cryptsetup", args)
			cmd := exec.Command("cryptsetup", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Fatalln(err)
			}

			// If password was provided, send it to cryptsetup.
			if encryptionPassword != "" {
				fmt.Fprintln(stdin, encryptionPassword)
			}

			// Run cryptsetup to decrypt drive and any error is fatal due to it preventing all required drives from mounting.
			err = cmd.Start()
			if err != nil {
				log.Fatalln(err)
			}

			err = cmd.Wait()
			if err != nil {
				log.Fatalln(err)
			}

			// If we cannot verify that its decrypted, then we need to stop as mount won't work.
			if _, err := os.Stat(dmPath); err != nil {
				log.Fatalln("Unable to decrypt:", mount.CryptName)
			}

			// Now that its decrypted, update the source path for mounting.
			mount.Source = dmPath
		}

		// If we're already mounted on this mountpoint, skip to the next one.
		if isMounted(mount.Target) {
			fmt.Println(mount.Target, "is already mounted")
			continue
		}

		// Mount the mountpoint.
		args := []string{
			"-t",
			mount.FSType,
			"-o",
			mount.Flags,
			mount.Source,
			mount.Target,
		}

		fmt.Println("mount", args)
		cmd := exec.Command("mount", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run mount to mount the mountpoint, any error is fatal as we want to ensure that mountpoints mount.
		err = cmd.Start()
		if err != nil {
			log.Fatalln(err)
		}

		err = cmd.Wait()
		if err != nil {
			log.Fatalln(err)
		}

		// Verified that it actually mounted.
		if !isMounted(mount.Target) {
			log.Fatalln("Unable to mount:", mount.Target)
			continue
		}
	}

	// Now that all mountpoints are mounted, start the services in configuration.
	for _, service := range app.config.Services {
		// Start the service.
		args := []string{
			"start",
			service,
		}

		fmt.Println("systemctl", args)
		cmd := exec.Command("systemctl", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Start systemctl, any error is not fatal to allow other services to start.
		err = cmd.Start()
		if err != nil {
			log.Println(err)
		}

		err = cmd.Wait()
		if err != nil {
			log.Println(err)
		}
	}
}
