package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	newRepoPath    = "/tmp/new_repo"
	oldRepoPath    = "/tmp/old_repo"
	mergedRepoPath = "/tmp/merged_repo"
	version        = "development"
)

type RpmCmdOpts struct {
	Bucket       string
	Prefix       string
	Visibility   string
	AwsAccessKey string
	AwsSecretKey string
	AwsRegion    string
	Sign         bool
	SignPass     string
	RpmFiles     []string
	Rebuild      bool
}

var rpmCmdOpts RpmCmdOpts

func signRepo(password, repoPath string) error {
	repomdPath := fmt.Sprintf("%s/repodata/repomd.xml", repoPath)

	if password != "" {
		command := fmt.Sprintf(`
expect -c '
set timeout 60
spawn gpg --pinentry-mode loopback --force-v3-sigs --verbose --detach-sign --armor %s
expect -re "Enter passphrase.*"
send -- "%s\r"
expect eof
lassign [wait] _ _ _ code
exit $code
'
`, repomdPath, password)

		logrus.Infof("Signing %s (interactive passphrase).", repomdPath)
		cmd := exec.Command("bash", "-c", command)
		return cmd.Run()
	} else {
		logrus.Infof("Signing %s without password", repomdPath)
		cmd := exec.Command("gpg", "--detach-sign", "--armor", repomdPath)
		return cmd.Run()
	}
}

func sign(password, rpmPath string) error {
	if password != "" {
		command := fmt.Sprintf(`
expect -c '
set timeout 60
spawn rpmsign --addsign %s
expect -re "Enter passphrase.*"
send -- "%s\r"
expect eof
lassign [wait] _ _ _ code
exit $code
'
`, rpmPath, password)
		cmd := exec.Command("bash", "-c", command)
		return cmd.Run()
	} else {
		logrus.Infof("Signing %s without password", rpmPath)
		cmd := exec.Command("rpmsign", "--addsign", rpmPath)
		return cmd.Run()
	}
}

func createS3Client(accessKey, secretKey, region string) (*s3.Client, error) {
	configOpts := []func(*config.LoadOptions) error{
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	}

	if region != "" {
		configOpts = append(configOpts, config.WithDefaultRegion(region))
	} else {
		configOpts = append(configOpts, config.WithDefaultRegion("us-east-1"))
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), configOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	return s3.NewFromConfig(cfg), nil
}

func uploadS3Object(client *s3.Client, bucket, key, localPath string, visibility string) error {
	ctx := context.TODO()

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer file.Close()

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	}

	if visibility == "public" {
		input.ACL = types.ObjectCannedACLPublicRead
	} else {
		input.ACL = types.ObjectCannedACLPrivate
	}

	_, err = client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload object %s: %w", key, err)
	}

	logrus.Infof("Uploaded %s -> s3://%s/%s", localPath, bucket, key)
	return nil
}

func uploadDirectory(client *s3.Client, bucket, prefix, localDir, visibility string, rebuild bool) error {
	return filepath.WalkDir(localDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(localDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if rebuild && strings.Contains(relativePath, ".rpm") {
			// Skip RPM files during rebuild to avoid unnecessary uploads
			logrus.Infof("Skipping upload of %s during rebuild", relativePath)
			return nil
		}

		s3Key := relativePath
		if prefix != "" {
			s3Key = prefix + "/" + s3Key
		}

		return uploadS3Object(client, bucket, s3Key, path, visibility)
	})
}

func deleteS3Objects(client *s3.Client, bucket string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	ctx := context.TODO()

	var objectIds []types.ObjectIdentifier
	for _, key := range keys {
		objectIds = append(objectIds, types.ObjectIdentifier{
			Key: aws.String(key),
		})
	}

	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &types.Delete{
			Objects: objectIds,
		},
	}

	result, err := client.DeleteObjects(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	logrus.Infof("Deleted %d objects from S3", len(result.Deleted))

	if len(result.Errors) > 0 {
		for _, deleteError := range result.Errors {
			logrus.Errorf("Failed to delete %s: %s", *deleteError.Key, *deleteError.Message)
		}
	}

	return nil
}

func listS3Objects(client *s3.Client, bucket, prefix string) ([]types.Object, error) {
	ctx := context.TODO()

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	var objects []types.Object

	paginator := s3.NewListObjectsV2Paginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}
		objects = append(objects, page.Contents...)
	}

	return objects, nil
}

func downloadS3Object(client *s3.Client, bucket, key, localPath string) error {
	ctx := context.TODO()

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := client.GetObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to download object %s: %w", key, err)
	}
	defer result.Body.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localPath, err)
	}
	defer file.Close()

	_, err = io.Copy(file, result.Body)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", localPath, err)
	}

	logrus.Infof("Downloaded %s -> %s", key, localPath)
	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func rebuild(client *s3.Client, rpms []types.Object, repodata []types.Object) error {
	logrus.Info("Rebuild mode enabled. Clearing old, new, and merged repository directories.")

	if len(rpms) > 0 {
		logrus.Infof("Found %d items in S3 bucket %s with prefix %s", len(rpms), rpmCmdOpts.Bucket, rpmCmdOpts.Prefix)
		for _, item := range rpms {
			relativePath := *item.Key

			// this is for when prefix is set, mostly to avoid the need to handle newRepoPath+prefix+"/"+file in newRepoPath
			if rpmCmdOpts.Prefix != "" {
				relativePath = (*item.Key)[len(rpmCmdOpts.Prefix)+1:]
			}

			localPath := filepath.Join(newRepoPath, relativePath)
			if err := downloadS3Object(client, rpmCmdOpts.Bucket, *item.Key, localPath); err != nil {
				return err
			}
		}

		logrus.Info("Old RPMs downloaded from S3.")
		logrus.Info("Running createrepo_c to rebuild RPMs.")
		comd := exec.Command("createrepo_c", "--checksum", "sha256", newRepoPath)
		if err := comd.Run(); err != nil {
			return err
		}

		if rpmCmdOpts.Sign {
			logrus.Info("Signing new repository metadata.")
			if err := signRepo(rpmCmdOpts.SignPass, newRepoPath); err != nil {
				return err
			}
		}

		// first we upload and after that the tool will delete the unnecessary items that we got in the list
		if err := uploadDirectory(client, rpmCmdOpts.Bucket, rpmCmdOpts.Prefix, newRepoPath, rpmCmdOpts.Visibility, rpmCmdOpts.Rebuild); err != nil {
			return err
		}

		var keysToDelete []string
		for _, item := range repodata {
			if strings.Contains(*item.Key, "repomd") {
				logrus.Infof("Skipping deletion of repomd file: %s", *item.Key)
				continue
			}

			keysToDelete = append(keysToDelete, *item.Key)
		}

		logrus.Infof("Deleting old repodata files from S3: %v", keysToDelete)
		if err := deleteS3Objects(client, rpmCmdOpts.Bucket, keysToDelete); err != nil {
			return err
		}

	} else {
		logrus.Info("No existing RPMs found in S3.")
	}

	return nil
}

func mergerepo(client *s3.Client, repodata []types.Object, newRpms []string) error {
	logrus.Infof("Found %d items in S3 bucket %s with prefix %s", len(repodata), rpmCmdOpts.Bucket, rpmCmdOpts.Prefix+"repodata")
	for _, item := range repodata {
		localPath := filepath.Join(oldRepoPath, "repodata")
		itemPath := filepath.Join(localPath, filepath.Base(*item.Key))
		if err := downloadS3Object(client, rpmCmdOpts.Bucket, *item.Key, itemPath); err != nil {
			return err
		}
	}

	logrus.Info("Creating mergepo folder to merge old + new repos")
	if err := os.MkdirAll(mergedRepoPath, 0777); err != nil {
		return err
	}

	mergeRepoScriptCmd := exec.Command("mergerepo_c",
		"--repo="+oldRepoPath,
		"--repo="+newRepoPath,
		"--all",
		"--omit-baseurl",
		"-o", mergedRepoPath)

	if err := mergeRepoScriptCmd.Run(); err != nil {
		return fmt.Errorf("failed to merge repositories: %w", err)
	}

	repodataMerged := filepath.Join(mergedRepoPath, "repodata")
	repomdMerged := filepath.Join(repodataMerged, "repomd.xml")

	logrus.Infof("Merged repodata created at: %s", repodataMerged)
	logrus.Infof("Merged repomd.xml location: %s", repomdMerged)

	if rpmCmdOpts.Sign {
		logrus.Info("Signing merged repository metadata.")
		if err := signRepo(rpmCmdOpts.SignPass, mergedRepoPath); err != nil {
			return err
		}
	}

	// first we upload and after that the tool will delete the unnecessary items that we got in the list
	if err := uploadDirectory(client, rpmCmdOpts.Bucket, rpmCmdOpts.Prefix, mergedRepoPath, rpmCmdOpts.Visibility, rpmCmdOpts.Rebuild); err != nil {
		return err
	}

	for _, rpm := range newRpms {
		s3Key := filepath.Base(rpm)
		if rpmCmdOpts.Prefix != "" {
			s3Key = rpmCmdOpts.Prefix + "/" + s3Key
		}

		if err := uploadS3Object(client, rpmCmdOpts.Bucket, s3Key, rpm, rpmCmdOpts.Visibility); err != nil {
			return err
		}
	}

	var keysToDelete []string
	for _, item := range repodata {
		if strings.Contains(*item.Key, "repomd") {
			logrus.Infof("Skipping deletion of repomd file: %s", *item.Key)
			continue
		}

		keysToDelete = append(keysToDelete, *item.Key)
	}

	logrus.Infof("Deleting old repodata files from S3: %v", keysToDelete)
	if err := deleteS3Objects(client, rpmCmdOpts.Bucket, keysToDelete); err != nil {
		return err
	}

	return nil
}

func rpmTool(cmd *cobra.Command, args []string) error {
	rpmCmdOpts.RpmFiles = args

	client, err := createS3Client(rpmCmdOpts.AwsAccessKey, rpmCmdOpts.AwsSecretKey, rpmCmdOpts.AwsRegion)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(newRepoPath, 0777); err != nil {
		return err
	}

	items, err := listS3Objects(client, rpmCmdOpts.Bucket, rpmCmdOpts.Prefix)
	if err != nil {
		return err
	}

	var repodata []types.Object
	var rpms []types.Object

	for _, item := range items {
		logrus.Infof("Found existing item in S3: %s", *item.Key)
		if strings.Contains(*item.Key, "/repodata/") {
			repodata = append(repodata, item)
		} else {
			rpms = append(rpms, item)
		}
	}

	if err := os.MkdirAll(oldRepoPath, 0777); err != nil {
		return err
	}

	if rpmCmdOpts.Rebuild {
		return rebuild(client, rpms, repodata)
	}

	if len(rpmCmdOpts.RpmFiles) == 0 {
		return errors.New("at least one RPM file must be provided")
	}

	var newRpms []string
	for _, rpmFile := range rpmCmdOpts.RpmFiles {
		// verify if the RPM already exists in S3 and stops the process if it does
		basename := filepath.Base(rpmFile)
		for _, rpm := range rpms {
			if filepath.Base(*rpm.Key) == basename {
				return fmt.Errorf("RPM %s already exists in S3 bucket %s with prefix %s", basename, rpmCmdOpts.Bucket, rpmCmdOpts.Prefix)
			}
		}

		localDest := filepath.Join(newRepoPath, basename)
		logrus.Infof("Copying %s to %s", rpmFile, localDest)
		newRpms = append(newRpms, localDest)
		if err := copyFile(rpmFile, localDest); err != nil {
			return err
		}

		// Sign RPMs if the sign flag is set
		if rpmCmdOpts.Sign {
			logrus.Infof("Signing %s", localDest)
			if err := sign(rpmCmdOpts.SignPass, localDest); err != nil {
				return err
			}

			// Verify the signature
			cmd := exec.Command("rpm", "-qpi", localDest)
			output, err := cmd.Output()
			if err == nil {
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")

				logrus.Info("=== RPM Package Info ===")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						if strings.Contains(line, "Name") ||
							strings.Contains(line, "Version") ||
							strings.Contains(line, "Signature") ||
							strings.Contains(line, "Build Date") {
							logrus.Infof("  %s", line)
						}
					}
				}
				logrus.Info("=== End RPM Info ===")

				if strings.Contains(string(output), "Signature") {
					logrus.Infof("RPM %s successfully signed", filepath.Base(localDest))
				} else {
					logrus.Warnf("RPM %s may not be properly signed", filepath.Base(localDest))
				}
			}
		}
	}

	logrus.Info("Running createrepo_c for new RPMs only.")
	comd := exec.Command("createrepo_c", "--checksum", "sha256", newRepoPath)
	if err := comd.Run(); err != nil {
		return err
	}

	repodataNew := filepath.Join(newRepoPath, "repodata")
	repomdNew := filepath.Join(repodataNew, "repomd.xml")

	logrus.Infof("Repodata created at: %s", repodataNew)
	logrus.Infof("Repomd.xml location: %s", repomdNew)

	if len(repodata) > 0 {
		return mergerepo(client, repodata, newRpms)
	}

	logrus.Info("No existing repodata found in S3. Uploading new RPMs and repodata.")

	if rpmCmdOpts.Sign {
		logrus.Info("Signing new repository metadata.")
		if err := signRepo(rpmCmdOpts.SignPass, newRepoPath); err != nil {
			return err
		}
	}

	return uploadDirectory(client, rpmCmdOpts.Bucket, rpmCmdOpts.Prefix, newRepoPath, rpmCmdOpts.Visibility, rpmCmdOpts.Rebuild)
}

func main() {
	cmd := &cobra.Command{
		Use:     "rpm",
		Short:   "Handle rpms in a S3 bucket",
		Long:    "The rpm is required to run in a OS/container with createrepo_c and mergerepo_c",
		RunE:    rpmTool,
		Version: version,
	}

	cmd.Flags().StringVarP(&rpmCmdOpts.Bucket, "bucket", "b", "", "S3 bucket")
	cmd.Flags().StringVarP(&rpmCmdOpts.Prefix, "prefix", "p", "", "S3 prefix")
	cmd.Flags().StringVar(&rpmCmdOpts.Visibility, "visibility", "private", "S3 ACL (default: \"private\")")
	cmd.Flags().StringVar(&rpmCmdOpts.AwsAccessKey, "aws-access-key", "", "AWS Access Key ID")
	cmd.Flags().StringVar(&rpmCmdOpts.AwsSecretKey, "aws-secret-key", "", "AWS Secret Access Key")
	cmd.Flags().BoolVar(&rpmCmdOpts.Sign, "sign", false, "Sign RPMs with rpmsign")
	cmd.Flags().StringVar(&rpmCmdOpts.AwsRegion, "aws-region", "us-east-1", "AWS Region")
	cmd.Flags().StringVar(&rpmCmdOpts.SignPass, "sign-pass", "", "Passphrase for signing (can be empty)")
	cmd.Flags().BoolVar(&rpmCmdOpts.Rebuild, "rebuild", false, "Rebuild the repository metadata")

	if err := cmd.MarkFlagRequired("bucket"); err != nil {
		logrus.Fatal(err)
	}

	if err := cmd.MarkFlagRequired("aws-access-key"); err != nil {
		logrus.Fatal(err)
	}

	if err := cmd.MarkFlagRequired("aws-secret-key"); err != nil {
		logrus.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
