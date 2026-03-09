<!-- markdownlint-disable-next-line first-line-heading no-inline-html -->
[<img src="https://rclone.org/img/logo_on_light__horizontal_color.svg" width="50%" alt="rclone logo">](https://rclone.org/#gh-light-mode-only)
<!-- markdownlint-disable-next-line no-inline-html -->
[<img src="https://rclone.org/img/logo_on_dark__horizontal_color.svg" width="50%" alt="rclone logo">](https://rclone.org/#gh-dark-mode-only)

[网站](https://rclone.org) |
[文档](https://rclone.org/docs/) |
[下载](https://rclone.org/downloads/) |
[贡献](CONTRIBUTING.md) |
[更新日志](https://rclone.org/changelog/) |
[安装](https://rclone.org/install/) |
[论坛](https://forum.rclone.org/)

[![构建状态](https://github.com/rclone/rclone/workflows/build/badge.svg)](https://github.com/rclone/rclone/actions?query=workflow%3Abuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/rclone/rclone)](https://goreportcard.com/report/github.com/rclone/rclone)
[![GoDoc](https://godoc.org/github.com/rclone/rclone?status.svg)](https://godoc.org/github.com/rclone/rclone)
[![Docker 拉取次数](https://img.shields.io/docker/pulls/rclone/rclone)](https://hub.docker.com/r/rclone/rclone)

# Rclone

Rclone *（“云存储版 rsync”）* 是一个命令行程序，用于在不同云存储提供商之间同步文件和目录。

## 存储提供商

- 1Fichier [:page_facing_up:](https://rclone.org/fichier/)
- Akamai Netstorage [:page_facing_up:](https://rclone.org/netstorage/)
- 阿里云（Aliyun）对象存储系统（OSS）[:page_facing_up:](https://rclone.org/s3/#alibaba-oss)
- Amazon S3 [:page_facing_up:](https://rclone.org/s3/)
- ArvanCloud 对象存储（AOS）[:page_facing_up:](https://rclone.org/s3/#arvan-cloud-object-storage-aos)
- Bizfly Cloud 简单存储 [:page_facing_up:](https://rclone.org/s3/#bizflycloud)
- Backblaze B2 [:page_facing_up:](https://rclone.org/b2/)
- Box [:page_facing_up:](https://rclone.org/box/)
- Ceph [:page_facing_up:](https://rclone.org/s3/#ceph/)
- 中国移动云弹性对象存储（EOS）[:page_facing_up:](https://rclone.org/s3/#china-mobile-ecloud-eos)
- Citrix ShareFile [:page_facing_up:](https://rclone.org/sharefile/)
- Cloudflare R2 [:page_facing_up:](https://rclone.org/s3/#cloudflare-r2)
- Cloudinary [:page_facing_up:](https://rclone.org/cloudinary/)
- Cubbit DS3 [:page_facing_up:](https://rclone.org/s3/#Cubbit)
- DigitalOcean Spaces [:page_facing_up:](https://rclone.org/s3/#digitalocean-spaces)
- Digi Storage [:page_facing_up:](https://rclone.org/koofr/#digi-storage)
- Dreamhost [:page_facing_up:](https://rclone.org/s3/#dreamhost)
- Drime [:page_facing_up:](https://rclone.org/s3/#drime)
- Dropbox [:page_facing_up:](https://rclone.org/dropbox/)
- Enterprise File Fabric [:page_facing_up:](https://rclone.org/filefabric/)
- Exaba [:page_facing_up:](https://rclone.org/s3/#exaba)
- Fastly 对象存储 [:page_facing_up:](https://rclone.org/s3/#fastly)
- Fastmail Files [:page_facing_up:](https://rclone.org/webdav/#fastmail-files)
- FileLu [:page_facing_up:](https://rclone.org/filelu/)
- Filen [:page_facing_up:](https://rclone.org/filen/)
- Files.com [:page_facing_up:](https://rclone.org/filescom/)
- FlashBlade [:page_facing_up:](https://rclone.org/s3/#pure-storage-flashblade)
- FTP [:page_facing_up:](https://rclone.org/ftp/)
- GoFile [:page_facing_up:](https://rclone.org/gofile/)
- Google Cloud Storage [:page_facing_up:](https://rclone.org/googlecloudstorage/)
- Google Drive [:page_facing_up:](https://rclone.org/drive/)
- Google Photos [:page_facing_up:](https://rclone.org/googlephotos/)
- HDFS（Hadoop 分布式文件系统）[:page_facing_up:](https://rclone.org/hdfs/)
- Hetzner 对象存储 [:page_facing_up:](https://rclone.org/s3/#hetzner)
- Hetzner Storage Box [:page_facing_up:](https://rclone.org/sftp/#hetzner-storage-box)
- HiDrive [:page_facing_up:](https://rclone.org/hidrive/)
- HTTP [:page_facing_up:](https://rclone.org/http/)
- 华为云对象存储服务（OBS）[:page_facing_up:](https://rclone.org/s3/#huawei-obs)
- iCloud Drive [:page_facing_up:](https://rclone.org/iclouddrive/)
- ImageKit [:page_facing_up:](https://rclone.org/imagekit/)
- Internet Archive [:page_facing_up:](https://rclone.org/internetarchive/)
- Internxt [:page_facing_up:](https://rclone.org/internxt/)
- Jottacloud [:page_facing_up:](https://rclone.org/jottacloud/)
- IBM COS S3 [:page_facing_up:](https://rclone.org/s3/#ibm-cos-s3)
- Intercolo 对象存储 [:page_facing_up:](https://rclone.org/s3/#intercolo)
- IONOS Cloud [:page_facing_up:](https://rclone.org/s3/#ionos)
- Koofr [:page_facing_up:](https://rclone.org/koofr/)
- Leviia 对象存储 [:page_facing_up:](https://rclone.org/s3/#leviia)
- Liara 对象存储 [:page_facing_up:](https://rclone.org/s3/#liara-object-storage)
- Linkbox [:page_facing_up:](https://rclone.org/linkbox)
- Linode 对象存储 [:page_facing_up:](https://rclone.org/s3/#linode)
- Magalu 对象存储 [:page_facing_up:](https://rclone.org/s3/#magalu)
- Mail.ru Cloud [:page_facing_up:](https://rclone.org/mailru/)
- Memset Memstore [:page_facing_up:](https://rclone.org/swift/)
- MEGA [:page_facing_up:](https://rclone.org/mega/)
- MEGA S4 对象存储 [:page_facing_up:](https://rclone.org/s3/#mega)
- Memory [:page_facing_up:](https://rclone.org/memory/)
- Microsoft Azure Blob Storage [:page_facing_up:](https://rclone.org/azureblob/)
- Microsoft Azure Files Storage [:page_facing_up:](https://rclone.org/azurefiles/)
- Microsoft OneDrive [:page_facing_up:](https://rclone.org/onedrive/)
- Minio [:page_facing_up:](https://rclone.org/s3/#minio)
- Nextcloud [:page_facing_up:](https://rclone.org/webdav/#nextcloud)
- Blomp Cloud Storage [:page_facing_up:](https://rclone.org/swift/)
- OpenDrive [:page_facing_up:](https://rclone.org/opendrive/)
- OpenStack Swift [:page_facing_up:](https://rclone.org/swift/)
- Oracle Cloud Storage [:page_facing_up:](https://rclone.org/swift/)
- Oracle 对象存储 [:page_facing_up:](https://rclone.org/oracleobjectstorage/)
- Outscale [:page_facing_up:](https://rclone.org/s3/#outscale)
- OVHcloud 对象存储（Swift）[:page_facing_up:](https://rclone.org/swift/)
- OVHcloud 对象存储（兼容 S3）[:page_facing_up:](https://rclone.org/s3/#ovhcloud)
- ownCloud [:page_facing_up:](https://rclone.org/webdav/#owncloud)
- pCloud [:page_facing_up:](https://rclone.org/pcloud/)
- Petabox [:page_facing_up:](https://rclone.org/s3/#petabox)
- PikPak [:page_facing_up:](https://rclone.org/pikpak/)
- Pixeldrain [:page_facing_up:](https://rclone.org/pixeldrain/)
- premiumize.me [:page_facing_up:](https://rclone.org/premiumizeme/)
- put.io [:page_facing_up:](https://rclone.org/putio/)
- Proton Drive [:page_facing_up:](https://rclone.org/protondrive/)
- QingStor [:page_facing_up:](https://rclone.org/qingstor/)
- 七牛云对象存储（Kodo）[:page_facing_up:](https://rclone.org/s3/#qiniu)
- Rabata Cloud Storage [:page_facing_up:](https://rclone.org/s3/#Rabata)
- Quatrix [:page_facing_up:](https://rclone.org/quatrix/)
- Rackspace Cloud Files [:page_facing_up:](https://rclone.org/swift/)
- RackCorp 对象存储 [:page_facing_up:](https://rclone.org/s3/#RackCorp)
- rsync.net [:page_facing_up:](https://rclone.org/sftp/#rsync-net)
- Scaleway [:page_facing_up:](https://rclone.org/s3/#scaleway)
- Seafile [:page_facing_up:](https://rclone.org/seafile/)
- Seagate Lyve Cloud [:page_facing_up:](https://rclone.org/s3/#lyve)
- SeaweedFS [:page_facing_up:](https://rclone.org/s3/#seaweedfs)
- Selectel 对象存储 [:page_facing_up:](https://rclone.org/s3/#selectel)
- Servercore 对象存储 [:page_facing_up:](https://rclone.org/s3/#servercore)
- SFTP [:page_facing_up:](https://rclone.org/sftp/)
- Shade [:page_facing_up:](https://rclone.org/shade/)
- SMB / CIFS [:page_facing_up:](https://rclone.org/smb/)
- Spectra Logic [:page_facing_up:](https://rclone.org/s3/#spectralogic)
- Storj [:page_facing_up:](https://rclone.org/storj/)
- SugarSync [:page_facing_up:](https://rclone.org/sugarsync/)
- Synology C2 对象存储 [:page_facing_up:](https://rclone.org/s3/#synology-c2)
- 腾讯云对象存储（COS）[:page_facing_up:](https://rclone.org/s3/#tencent-cos)
- Uloz.to [:page_facing_up:](https://rclone.org/ulozto/)
- Wasabi [:page_facing_up:](https://rclone.org/s3/#wasabi)
- WebDAV [:page_facing_up:](https://rclone.org/webdav/)
- Yandex Disk [:page_facing_up:](https://rclone.org/yandex/)
- Zadara 对象存储 [:page_facing_up:](https://rclone.org/s3/#zadara)
- Zoho WorkDrive [:page_facing_up:](https://rclone.org/zoho/)
- Zata.ai [:page_facing_up:](https://rclone.org/s3/#Zata)
- 本地文件系统 [:page_facing_up:](https://rclone.org/local/)

请参阅[所有存储提供商及其功能的完整列表](https://rclone.org/overview/)

### 虚拟存储提供商

这些后端会适配或修改其他存储提供商

- Alias：重命名现有远程存储 [:page_facing_up:](https://rclone.org/alias/)
- Archive：读取归档文件 [:page_facing_up:](https://rclone.org/archive/)
- Cache：缓存远程存储（已弃用）[:page_facing_up:](https://rclone.org/cache/)
- Chunker：拆分大文件 [:page_facing_up:](https://rclone.org/chunker/)
- Combine：将多个远程存储组合成目录树 [:page_facing_up:](https://rclone.org/combine/)
- Compress：压缩文件 [:page_facing_up:](https://rclone.org/compress/)
- Crypt：加密文件 [:page_facing_up:](https://rclone.org/crypt/)
- Hasher：计算哈希 [:page_facing_up:](https://rclone.org/hasher/)
- Union：将多个远程存储联合起来协同工作 [:page_facing_up:](https://rclone.org/union/)

## 功能

- 始终检查 MD5/SHA-1 哈希以确保文件完整性
- 保留文件时间戳
- 支持基于整文件的部分同步
- [Copy](https://rclone.org/commands/rclone_copy/) 模式，仅复制新增/变更的文件
- [Sync](https://rclone.org/commands/rclone_sync/)（单向）模式，使目录保持完全一致
- [Bisync](https://rclone.org/bisync/)（双向）模式，以双向方式保持两个目录同步
- [Check](https://rclone.org/commands/rclone_check/) 模式，用于检查文件哈希是否相等
- 可以在网络之间同步，例如两个不同的云账户之间
- 可选的大文件分块（[Chunker](https://rclone.org/chunker/)）
- 可选的透明压缩（[Compress](https://rclone.org/compress/)）
- 可选的加密（[Crypt](https://rclone.org/crypt/)）
- 可选的 FUSE 挂载（[rclone mount](https://rclone.org/commands/rclone_mount/)）
- 多线程下载到本地磁盘
- 可以通过 [serve](https://rclone.org/commands/rclone_serve/) 在 HTTP/WebDAV/FTP/SFTP/DLNA 上提供本地或远程文件服务

## 安装与文档

请访问 [rclone 网站](https://rclone.org/) 了解：

- [安装](https://rclone.org/install/)
- [文档与配置](https://rclone.org/docs/)
- [更新日志](https://rclone.org/changelog/)
- [常见问题](https://rclone.org/faq/)
- [存储提供商](https://rclone.org/overview/)
- [论坛](https://forum.rclone.org/)
- ……以及更多内容

## 下载

- <https://rclone.org/downloads/>

## 许可证

这是在 MIT 许可证条款下发布的自由软件（请查看本软件包中附带的 [COPYING 文件](/COPYING)）。
