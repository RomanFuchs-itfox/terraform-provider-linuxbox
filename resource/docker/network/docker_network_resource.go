package network

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alessio/shellescape"
	"github.com/draganm/terraform-provider-linuxbox/sshsession"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/pkg/errors"
)

func Resource() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreate,
		Read:   resourceRead,
		Update: resourceUpdate,
		Delete: resourceDelete,

		Schema: map[string]*schema.Schema{
			"ssh_key": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},

			"ssh_user": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Default:  "root",
				Optional: true,
			},

			"host_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCreate(d *schema.ResourceData, m interface{}) error {

	ssh, err := sshsession.Open(d)
	if err != nil {
		return errors.Wrap(err, "while creating ssh session")
	}

	defer ssh.Close()

	name := d.Get("name").(string)

	cmd := []string{
		"docker",
		"network",
		"create",
		shellescape.Quote(name),
	}

	line := strings.Join(cmd, " ")

	stdout, stderr, err := ssh.RunInSession(line)
	if err != nil {
		return errors.Wrapf(err, "while running `%s`: %s", line, string(stderr))
	}

	id := strings.TrimSuffix(string(stdout), "\n")

	d.SetId(id)

	return nil
}

func resourceRead(d *schema.ResourceData, m interface{}) error {

	ssh, err := sshsession.Open(d)
	if err != nil {
		return errors.Wrap(err, "while creating ssh session")
	}

	defer ssh.Close()

	stdout, _, err := ssh.RunInSession(fmt.Sprintf("docker network inspect %s", d.Id()))
	if sshsession.IsExecError(err) {
		d.SetId("")
		return nil
	}

	type network struct {
		ID   string `json:"Id"`
		Name string `json:"Name"`
	}

	networks := []network{}

	err = json.Unmarshal(stdout, &networks)

	if err != nil {
		return errors.Wrap(err, "while parsing docker network json")
	}

	if len(networks) != 1 {
		return errors.Errorf("expected one network with id %s, found %d", d.Id(), len(networks))
	}

	d.Set("name", networks[0].Name)

	return nil
}

func resourceUpdate(d *schema.ResourceData, m interface{}) error {
	return errors.New("update is not supported")
}

func resourceDelete(d *schema.ResourceData, m interface{}) error {
	ssh, err := sshsession.Open(d)
	if err != nil {
		return errors.Wrap(err, "while creating ssh session")
	}

	defer ssh.Close()

	_, _, err = ssh.RunInSession(fmt.Sprintf("docker network rm %s", d.Id()))

	return err
}