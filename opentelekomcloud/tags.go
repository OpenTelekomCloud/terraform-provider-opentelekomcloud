package opentelekomcloud

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	//"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ec2"
	//"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// tagsSchema returns the schema to use for tags.
//
func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
	}
}

func tagsSchemaComputed() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Computed: true,
	}
}

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTags(conn *ec2.EC2, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTags(tagsFromMap(o), tagsFromMap(n))

		// Set tags
		if len(remove) > 0 {
			err := resource.Retry(5*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] Removing tags: %#v from %s", remove, d.Id())
				_, err := conn.DeleteTags(&ec2.DeleteTagsInput{
					Resources: []*string{aws.String(d.Id())},
					Tags:      remove,
				})
				if err != nil {
					ec2err, ok := err.(awserr.Error)
					if ok && strings.Contains(ec2err.Code(), ".NotFound") {
						return resource.RetryableError(err) // retry
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			err := resource.Retry(5*time.Minute, func() *resource.RetryError {
				log.Printf("[DEBUG] Creating tags: %s for %s", create, d.Id())
				_, err := conn.CreateTags(&ec2.CreateTagsInput{
					Resources: []*string{aws.String(d.Id())},
					Tags:      create,
				})
				if err != nil {
					ec2err, ok := err.(awserr.Error)
					if ok && strings.Contains(ec2err.Code(), ".NotFound") {
						return resource.RetryableError(err) // retry
					}
					return resource.NonRetryableError(err)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTags(oldTags, newTags []*ec2.Tag) ([]*ec2.Tag, []*ec2.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[*t.Key] = *t.Value
	}

	// Build the list of what to remove
	var remove []*ec2.Tag
	for _, t := range oldTags {
		old, ok := create[*t.Key]
		if !ok || old != *t.Value {
			remove = append(remove, t)
		}
	}

	return tagsFromMap(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMap(m map[string]interface{}) []*ec2.Tag {
	result := make([]*ec2.Tag, 0, len(m))
	for k, v := range m {
		t := &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnored(t) {
			result = append(result, t)
		}
	}

	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMap(ts []*ec2.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnored(t) {
			result[*t.Key] = *t.Value
		}
	}

	return result
}

// tagIgnored compares a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnored(t *ec2.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, *t.Key)
		if r, _ := regexp.MatchString(v, *t.Key); r == true {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", *t.Key, *t.Value)
			return true
		}
	}
	return false
}
