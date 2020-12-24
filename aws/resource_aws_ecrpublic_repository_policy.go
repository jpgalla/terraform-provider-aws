package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecrpublic"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAwsEcrPublicRepositoryPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcrPublicRepositoryPolicyCreate,
		Read:   resourceAwsEcrPublicRepositoryPolicyRead,
		Update: resourceAwsEcrPublicRepositoryPolicyUpdate,
		Delete: resourceAwsEcrPublicRepositoryPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"policy": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: suppressEquivalentAwsPolicyDiffs,
			},
			"registry_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEcrPublicRepositoryPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrpublicconn

	input := ecrpublic.SetRepositoryPolicyInput{
		RepositoryName: aws.String(d.Get("repository").(string)),
		PolicyText:     aws.String(d.Get("policy").(string)),
	}

	log.Printf("[DEBUG] Creating ECR Public repository policy: %s", input)

	// Retry due to IAM eventual consistency
	var err error
	var out *ecrpublic.SetRepositoryPolicyOutput
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		out, err = conn.SetRepositoryPolicy(&input)

		if isAWSErr(err, "InvalidParameterException", "Invalid repository policy provided") {
			return resource.RetryableError(err)
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		out, err = conn.SetRepositoryPolicy(&input)
	}
	if err != nil {
		return fmt.Errorf("Error creating ECR Repository Policy: %s", err)
	}

	repositoryPolicy := *out

	log.Printf("[DEBUG] ECR Public repository policy created: %s", *repositoryPolicy.RepositoryName)

	d.SetId(aws.StringValue(repositoryPolicy.RepositoryName))
	d.Set("registry_id", repositoryPolicy.RegistryId)

	return resourceAwsEcrPublicRepositoryPolicyRead(d, meta)
}

func resourceAwsEcrPublicRepositoryPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrpublicconn

	log.Printf("[DEBUG] Reading repository policy %s", d.Id())
	out, err := conn.GetRepositoryPolicy(&ecrpublic.GetRepositoryPolicyInput{
		RepositoryName: aws.String(d.Id()),
	})
	if err != nil {
		if ecrerr, ok := err.(awserr.Error); ok {
			switch ecrerr.Code() {
			case "RepositoryNotFoundException", "RepositoryPolicyNotFoundException":
				d.SetId("")
				return nil
			default:
				return err
			}
		}
		return err
	}

	log.Printf("[DEBUG] Received repository policy %s", out)

	repositoryPolicy := out

	d.SetId(aws.StringValue(repositoryPolicy.RepositoryName))
	d.Set("repository", repositoryPolicy.RepositoryName)
	d.Set("registry_id", repositoryPolicy.RegistryId)
	d.Set("policy", repositoryPolicy.PolicyText)

	return nil
}

func resourceAwsEcrPublicRepositoryPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrpublicconn

	if !d.HasChange("policy") {
		return nil
	}

	input := ecrpublic.SetRepositoryPolicyInput{
		RepositoryName: aws.String(d.Get("repository").(string)),
		RegistryId:     aws.String(d.Get("registry_id").(string)),
		PolicyText:     aws.String(d.Get("policy").(string)),
	}

	log.Printf("[DEBUG] Updating ECR Public repository policy: %s", input)

	// Retry due to IAM eventual consistency
	var err error
	var out *ecrpublic.SetRepositoryPolicyOutput
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		out, err = conn.SetRepositoryPolicy(&input)

		if isAWSErr(err, "InvalidParameterException", "Invalid repository policy provided") {
			return resource.RetryableError(err)
		}
		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if isResourceTimeoutError(err) {
		out, err = conn.SetRepositoryPolicy(&input)
	}
	if err != nil {
		return fmt.Errorf("Error updating ECR Repository Policy: %s", err)
	}

	repositoryPolicy := *out

	d.SetId(aws.StringValue(repositoryPolicy.RepositoryName))
	d.Set("registry_id", repositoryPolicy.RegistryId)

	return nil
}

func resourceAwsEcrPublicRepositoryPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrpublicconn

	_, err := conn.DeleteRepositoryPolicy(&ecrpublic.DeleteRepositoryPolicyInput{
		RepositoryName: aws.String(d.Id()),
		RegistryId:     aws.String(d.Get("registry_id").(string)),
	})
	if err != nil {
		if ecrerr, ok := err.(awserr.Error); ok {
			switch ecrerr.Code() {
			case "RepositoryNotFoundException", "RepositoryPolicyNotFoundException":
				return nil
			default:
				return err
			}
		}
		return err
	}

	log.Printf("[DEBUG] repository policy %s deleted.", d.Id())

	return nil
}
