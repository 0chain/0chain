This guide contains step-by-step instructions for installing software dependencies related to Züs 

- [Install WSL with Docker](#install-wsl-with-docker)
  - [Cleanup](#cleanup)
- [Install Docker Desktop](#install-docker-desktop)
- [Install Go](#install-go)

## Install WSL with Docker.

1. Install Windows Subsystem for Linux (WSL) from the Microsoft Store.

![windows subsystem for linux](https://github.com/0chain/0chain/assets/65766301/043d363c-2d67-450f-aeb6-768cbd49f246)

2. Install a WSL2 distribution such as Ubuntu from the Microsoft Store.

![ubuntu from store](https://github.com/0chain/0chain/assets/65766301/c7506147-d686-438c-9942-6194d258a7a2)

3. After successfully installing Ubuntu, open the distribution through the Windows Store.

![wsl ubuntu open](https://github.com/0chain/0chain/assets/65766301/949782ae-f913-4ae4-b3d5-e7577bb80e5a)

4. A command prompt will appear, prompting you to enter your Unix username and password.

![ubuntu prompt username ](https://github.com/0chain/0chain/assets/65766301/2f27e22c-402e-49f6-ac25-31620ab0ae5a)

5. Install Docker Desktop for Windows forom [here](https://www.docker.com/products/docker-desktop/).

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

1. In Windows Settings, go to Add or Remove Programs, find Windows Subsystem for Linux, click on the app name, select three dots, and choose Uninstall
  ![uninstall windows subsystem for linux](https://github.com/0chain/0chain/assets/65766301/37354484-a49b-4c89-bb9f-c8b76d51ada1)

2. In Windows Settings, go to Add or Remove Programs, find Ubuntu, click on the app name, select three dots, and choose Uninstall.

![Screenshot 2023-11-22 132007](https://github.com/0chain/0chain/assets/65766301/85cdf294-4156-48ca-ae0e-ea99271d71b1)

3. In Windows Settings, go to Add or Remove Programs, find Docker Desktop, click on the app name, select three dots, and choose Uninstall.
![Screenshot (45)](https://github.com/0chain/0chain/assets/65766301/3285ccd3-ea4b-42c3-b8ad-6deab770f3f2)
