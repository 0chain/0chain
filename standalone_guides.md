This guide contains step-by-step instructions for installing software dependencies related to Züs. 

- [Install WSL with Docker](#install-wsl-with-docker)
  - [Cleanup](#cleanup)
- [Install Docker Desktop](#install-docker-desktop)
   - [Linux Installation](#linux-installation)
   - [Mac Installation](#mac-installation)
   - [Windows Installation](#windows-installation)
- [Install Go](#install-go)
   - [Go Linux Installation ](#go-linux-installation)
   - [Go Mac Installation](#go-mac-installation)
   - [Go Windows Installation](#go-windows-installation)
    

## Install WSL with Docker

1. Install Windows Subsystem for Linux (WSL) from the Microsoft Store.

![windows subsystem for linux](https://github.com/0chain/0chain/assets/65766301/043d363c-2d67-450f-aeb6-768cbd49f246)

2. Install a WSL2 distribution such as Ubuntu from the Microsoft Store.

![ubuntu from store](https://github.com/0chain/0chain/assets/65766301/c7506147-d686-438c-9942-6194d258a7a2)

3. After successfully installing Ubuntu, open the distribution through the Windows Store.

![wsl ubuntu open](https://github.com/0chain/0chain/assets/65766301/949782ae-f913-4ae4-b3d5-e7577bb80e5a)

4. A command prompt will appear, prompting you to enter your Unix username and password.

![ubuntu prompt username ](https://github.com/0chain/0chain/assets/65766301/2f27e22c-402e-49f6-ac25-31620ab0ae5a)

5. Install Docker Desktop for Windows from [here](https://www.docker.com/products/docker-desktop/).

![docker desktop for windows](https://github.com/0chain/0chain/assets/65766301/faacf49d-4944-4aba-a106-1caa224cb1b4)

Note: Restart your Windows machine after Docker installation.

6. Once installed, initiate Docker Desktop from the Windows Start menu.

![docker desktop](https://github.com/0chain/0chain/assets/65766301/7219d948-214c-408b-941c-18e12d2e737c)

7. Access the Settings pane at the top right of Docker Desktop.

![docker desktop settings](https://github.com/0chain/0chain/assets/65766301/a443892b-c125-4f08-980d-c7bac6d5cc96)

8. Navigate to Settings > Resources > WSL Integration. Toggle the switch for the Ubuntu distribution, then click on Apply and Restart.

![apply and restart](https://github.com/0chain/0chain/assets/65766301/e35216de-8952-485f-8293-0aeaf56e2980)

9. To confirm that Docker has been installed, open a WSL distribution (e.g. Ubuntu) and display the version and build number by entering: `docker --version`

![docker version](https://github.com/0chain/0chain/assets/65766301/2a1dd930-6801-4ccd-9071-8ee06fc69a1b)

### Cleanup

1. In Windows Settings, go to Add or Remove Programs, find Windows Subsystem for Linux, click on the app name, select three dots, and choose Uninstall.
  
![uninstall windows subsystem for linux](https://github.com/0chain/0chain/assets/65766301/37354484-a49b-4c89-bb9f-c8b76d51ada1)

2. In Windows Settings, go to Add or Remove Programs, find Ubuntu, click on the app name, select three dots, and choose Uninstall.

![Screenshot 2023-11-22 132007](https://github.com/0chain/0chain/assets/65766301/85cdf294-4156-48ca-ae0e-ea99271d71b1)

3. In Windows Settings, go to Add or Remove Programs, find Docker Desktop, click on the app name, select three dots, and choose Uninstall.

![Screenshot (45)](https://github.com/0chain/0chain/assets/65766301/3285ccd3-ea4b-42c3-b8ad-6deab770f3f2)

## Install Docker Desktop

* [Linux Installation](#linux-installation)
* [Mac Installation](#mac-installation)
* [Windows Installation](#windows-installation)


### Linux Installation

1. To manually download the Debian binary file from the official docker website. You can use the wget command in terminal as shown below.

```
wget https://desktop.docker.com/linux/main/amd64/docker-desktop-4.25.2-amd64.deb

```

2. Once the file is downloaded, install Docker Desktop by running the following apt command:

```
sudo apt install ./docker-desktop-4.25.2-amd64.deb
apt-install-docker-desktop-debian-binary
```

3. Once Docker Desktop is installed, you can use the application manager to search and launch it as shown. Shortly after, the Docker Desktop GUI dashboard will launch. Setup and finally, you will land on the Docker Desktop home page as you can see below with instructions on how to get started with containers.

![Run-sample-container-docker-desktop-ubuntu-768x446](https://github.com/0chain/0chain/assets/65766301/aa856ea9-1d0b-449b-a8d4-8e8a2bc73cc8)

### Mac Installation

1. Docker Desktop for Mac has two versions their download links are mentioned below:

- [Docker Desktop for Mac with Apple Chip(M1/M2/M3)](https://desktop.docker.com/mac/main/arm64/Docker.dmg) 
- [Docker Desktop for Intel Chip](https://desktop.docker.com/mac/main/amd64/Docker.dmg)

2. Double-click Docker.dmg to open the installer, then drag the Docker icon to the Applications folder.

3. Double-click Docker.app in the Applications folder to start Docker.

4. The Docker menu displays the Docker Subscription Service Agreement.Select Accept to continue.

Note that Docker Desktop won't run if you do not agree to the terms. You can choose to accept the terms at a later date by opening Docker Desktop.

5. From the installation window, select either:

- Use recommended settings (Requires password). This let's Docker Desktop automatically set the necessary configuration settings.
- Use advanced settings. You can then set the location of the Docker CLI tools either in the system or user directory, enable the default Docker socket, and enable privileged port mapping. See Settings, for more information and how to set the location of the Docker CLI tools.

6. Select Finish. If you have applied any of the above configurations that require a password in step 5, enter your password to confirm your choice.

### Windows Installation

1. Docker Desktop for Windows can be downloaded from [here](https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe).

2. Double-click Docker Desktop Installer.exe to run the installer.

3. Follow the instructions on the installation wizard to authorize the installer and proceed with the install.

4. When the installation is successful, select Close to complete the installation process.

5. Search for Docker app in windows start menu, and select Docker Desktop in the search results.

6. The Docker menu displays the Docker Subscription Service Agreement.Select Accept to continue. 

7. Docker Desktop starts after you accept the terms.

## Install Go

- [Go Linux Installation ](#linux-installation)
- [Go Mac Installation](#mac-installation)
- [Go Windows Installation](#windows-installation)


### Go Linux Installation

1. Download the Go linux tar package from [here](https://go.dev/dl/go1.21.4.linux-amd64.tar.gz).

2. Remove any previous Go installation(if any) by deleting the /usr/local/go folder, then extract the archive you just downloaded into /usr/local, creating a fresh Go tree in /usr/local/go:

```
 rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.4.linux-amd64.tar.gz
```
(You may need to run the command as root or through sudo).

3. Add /usr/local/go/bin to the PATH environment variable. You can do this by adding the following line to your `$HOME/.profile` or `/etc/profile` (for a system-wide installation):
```
export PATH=$PATH:/usr/local/go/bin
```
Note: Changes made to a profile file may not apply until the next time you log into your computer. To apply the changes immediately, just run the shell commands directly or execute them from the profile using a command such as source `$HOME/.profile`.

4. Verify that you've installed Go by opening a command prompt and typing the following command:
```
go version
```
### Go Mac Installation

1. Download the [Go for apple silicon mac](https://go.dev/dl/go1.21.4.darwin-arm64.pkg) or [Go for mac intel chip](https://go.dev/dl/go1.21.4.darwin-amd64.pkg) depending upon your requirements. 

2. Open the package file you downloaded and follow the prompts to install Go.The package installs the Go distribution to `/usr/local/go`. The package should put the `/usr/local/go/bin` directory in your PATH environment variable. You may need to restart any open terminal sessions for the change to take effect.

3. Verify that you've installed Go by opening a command prompt and typing the following command:
```
go version
```
The command above should print the installed version of Go.

### Go Windows Installation

1. Download the Go windows installer from [here](https://go.dev/dl/go1.21.4.windows-amd64.msi).

2. Open the MSI file you downloaded and follow the prompts to install Go.

**Note:** By default, the installer will install Go to Program Files or Program Files (x86). You can change the location as needed. After installing, you will need to close and reopen any open command prompts so that changes to the environment made by the installer are reflected at the command prompt.

3. Verify that you've installed Go.
  - In Windows, click the Start menu.
  - In the menu's search box, type cmd, then press the Enter key.
  - In the Command Prompt window that appears, type the following command:
  ```
    go version
  ```
  The command above should print the installed version of Go.
