package main

import (
	"errors"
	"fmt"
	"math/rand"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	flags "github.com/jessevdk/go-flags"
	"github.com/miekg/dns"
)

func main() {

	var options struct {
		AccessKeyID     *string `long:"access-key-id" description:"AWS Access Key Id" env:"AWS_ACCESS_KEY_ID" required:"true"`
		SecretAccessKey *string `long:"secret-access-key" description:"AWS Secret Access Key" env:"AWS_SECRET_ACCESS_KEY" required:"true"`
		HostedZoneID    *string `long:"hosted-zone-id" description:"AWS Host Zone ID" required:"true" env:"AWS_HOSTED_ZONE_ID" required:"true"`
		IpAddress       *string `long:"ip" description:"ip address"`
		Hostname        *string `long:"hostname" description:"hostname" required:"true" env:"HOSTNAME"`
		TTL             *int64  `long:"ttl" description:"record ttl" default:"60"`
	}

	_, err := flags.ParseArgs(&options, os.Args)
	if err != nil {
		os.Exit(1)
	}

	if options.IpAddress == nil {
		var e error
		options.IpAddress, e = LookupIpAddress()
		if e != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	credentials := credentials.NewStaticCredentials(
		*options.AccessKeyID,
		*options.SecretAccessKey,
		"",
	)

	svc := route53.New(session.New(), aws.NewConfig().WithCredentials(credentials))
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{ // Required
			Changes: []*route53.Change{ // Required
				{ // Required
					Action: aws.String("UPSERT"), // Required
					ResourceRecordSet: &route53.ResourceRecordSet{ // Required
						Name: options.Hostname, // Required
						Type: aws.String("A"),  // Required
						ResourceRecords: []*route53.ResourceRecord{
							{ // Required
								Value: options.IpAddress, // Required
							},
						},
						TTL: options.TTL,
					},
				},
			},
			Comment: aws.String("DynDNS update"),
		},
		HostedZoneId: options.HostedZoneID,
	}

	_, err = svc.ChangeResourceRecordSets(params)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func LookupIpAddress() (*string, error) {
	servers := []string{"resolver1.opendns.com", "resolver2.opendns.com", "resolver3.opendns.com", "resolver4.opendns.com"}
	server := servers[rand.Intn(len(servers))]
	client := new(dns.Client)
	message := new(dns.Msg)
	message.SetQuestion("myip.opendns.com.", dns.TypeA)
	message.RecursionDesired = false

	r, _, err := client.Exchange(message, server+":53")
	if err != nil {
		return nil, err
	}
	if r.Rcode != dns.RcodeSuccess {
		return nil, errors.New("Bad response")
	}
	for _, a := range r.Answer {
		if value, ok := a.(*dns.A); ok {
			return aws.String(value.A.String()), nil
		}
	}
	return nil, errors.New("Not found")
}
