# Raid Mount

This tool was designed to make it easy to mount encrypted hard drives for a snapraid/mergerfs configuration after a boot of a system. This allows your boot drive to be unencrypted so it can boot without intervention, which you can then finish the boot process via ssh remotely to mount encrypted drives and start services that use them. This does not fully protect your system against physical attack, but it is a compromise I am willing to work with on my system to allow me to finish a boot process if I were to be unable to access the system physically.

# Raid Mountpoint Table Format

The format of the raidtab file is similar to the fstab format, but instead of having dump/pass options, there is a CryptName option to specify the name of the device once unencrypted. The CryptName field must be unique per each encrypted drive, and cannot match any existing `/dev/mapper/` device name. If the CryptName is `none`, raid-mount will treat it as an unencrypted mount. There is an example file provided to make this concept easier to understand.

# Raid Mount Configuration

Simply create a directory as `/etc/raid-mount/` and place a `config.json` and `raidtab` file within this directory. Configuration options for `config.json` can be viewed in the `config.go` file and the `config.example.json` file.