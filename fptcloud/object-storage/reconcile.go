package fptcloud_object_storage

import (
	"encoding/json"
	"reflect"
)

// Creates in this API are not idempotent. The backend commits before the
// response reaches the client, so a response lost in transit - a gateway 504 is
// the common case - leaves an object behind that Terraform never recorded. The
// next apply retries the same POST, the backend answers "already exists", and no
// amount of retrying resolves it: there is no state entry to destroy, and these
// resources have no importer.
//
// Every create therefore reconciles when the call reports failure: it asks the
// API what is actually there and adopts the object only when it matches the
// configuration. An object of the same name carrying different settings is a
// real conflict and is reported as one, so a timed-out create is recovered
// without silently taking ownership of somebody else's object.
//
// Access keys are deliberately excluded. Their secret is returned only by the
// create call and cannot be read back, so an adopted key would be recorded
// without a usable secret - worse than a clean failure.

// createOutcome is the verdict of a read-back performed after a create reported
// failure.
type createOutcome int

const (
	// createFailed - nothing matching is on the server, so the original error
	// stands and is reported unchanged.
	createFailed createOutcome = iota
	// createAdopted - the object described by the configuration is present, so
	// the create had in fact succeeded and the resource is recorded as normal.
	createAdopted
	// createConflict - an object of that name exists but does not match the
	// configuration, so it is not ours to take over.
	createConflict
)

// reconcileBucket adopts a bucket only when it is empty. A bucket that already
// holds objects is somebody else's: adopting it would place data under Terraform
// management that a later destroy would delete. ListBuckets does not report acl
// or versioning, so emptiness is the only signal available here.
func reconcileBucket(service ObjectStorageService, vpcId, s3ServiceId, name string) createOutcome {
	buckets := service.ListBuckets(vpcId, s3ServiceId, 1, 1000)
	for _, bucket := range buckets.Buckets {
		if bucket.Name != name {
			continue
		}
		if bucket.IsEmpty {
			return createAdopted
		}
		return createConflict
	}
	return createFailed
}

// reconcileLifeCycleRule compares every field the rule declares against the
// stored rule. Fields the configuration omits are not compared, because the
// backend fills its own defaults for them.
func reconcileLifeCycleRule(service ObjectStorageService, vpcId, s3ServiceId, bucketName string, want S3BucketLifecycleConfig) createOutcome {
	response := service.GetBucketLifecycle(vpcId, s3ServiceId, bucketName, 1, maxRulesPerPage)
	if !response.Status {
		return createFailed
	}
	for _, rule := range response.Rules {
		if rule.ID != want.ID {
			continue
		}
		if want.Status != "" && rule.Status != want.Status {
			return createConflict
		}
		if want.Filter != nil && rule.Filter.Prefix != want.Filter.Prefix {
			return createConflict
		}
		if want.Expiration != nil &&
			(rule.Expiration.Days != want.Expiration.Days ||
				rule.Expiration.ExpiredObjectDeleteMarker != want.Expiration.ExpiredObjectDeleteMarker) {
			return createConflict
		}
		if want.NoncurrentVersionExpiration != nil &&
			rule.NoncurrentVersionExpiration.NoncurrentDays != want.NoncurrentVersionExpiration.NoncurrentDays {
			return createConflict
		}
		if want.AbortIncompleteMultipartUpload != nil &&
			rule.AbortIncompleteMultipartUpload.DaysAfterInitiation != want.AbortIncompleteMultipartUpload.DaysAfterInitiation {
			return createConflict
		}
		return createAdopted
	}
	return createFailed
}

// reconcileCorsRule compares the stored rule against the one the configuration
// asks for. Header lists are optional, so they are compared only when declared.
func reconcileCorsRule(service ObjectStorageService, vpcId, s3ServiceId, bucketName string, want CorsRule) createOutcome {
	response, err := service.GetBucketCors(vpcId, s3ServiceId, bucketName, 1, maxRulesPerPage)
	if err != nil || response == nil || !response.Status {
		return createFailed
	}
	for _, rule := range response.CorsRules {
		if rule.ID != want.ID {
			continue
		}
		if !equalStrings(rule.AllowedMethods, want.AllowedMethods) ||
			!equalStrings(rule.AllowedOrigins, want.AllowedOrigins) ||
			rule.MaxAgeSeconds != want.MaxAgeSeconds {
			return createConflict
		}
		if len(want.AllowedHeaders) > 0 && !equalStrings(rule.AllowedHeaders, want.AllowedHeaders) {
			return createConflict
		}
		if len(want.ExposeHeaders) > 0 && !equalStrings(rule.ExposeHeaders, want.ExposeHeaders) {
			return createConflict
		}
		return createAdopted
	}
	return createFailed
}

// reconcileBucketPolicy compares the stored document with the configured one
// semantically, so formatting differences do not read as a conflict.
func reconcileBucketPolicy(service ObjectStorageService, vpcId, s3ServiceId, bucketName, want string) createOutcome {
	response := service.GetBucketPolicy(vpcId, s3ServiceId, bucketName)
	if response == nil || !response.Status || response.Policy == "" {
		return createFailed
	}
	if equalJSON(response.Policy, want) {
		return createAdopted
	}
	return createConflict
}

// reconcileBucketAcl treats the desired canned ACL already being in place as
// success: PutBucketAcl overwrites, so there is nothing else to own.
func reconcileBucketAcl(service ObjectStorageService, vpcId, s3ServiceId, bucketName, want string) createOutcome {
	response := service.GetBucketAcl(vpcId, s3ServiceId, bucketName)
	if response == nil || !response.Status {
		return createFailed
	}
	if response.CannedACL == want {
		return createAdopted
	}
	return createFailed
}

// reconcileBucketVersioning mirrors reconcileBucketAcl: the setting is a single
// overwritable value, so finding it already applied means the call landed.
func reconcileBucketVersioning(service ObjectStorageService, vpcId, s3ServiceId, bucketName, want string) createOutcome {
	response := service.GetBucketVersioning(vpcId, s3ServiceId, bucketName)
	if response == nil || !response.Status {
		return createFailed
	}
	if response.Config == want {
		return createAdopted
	}
	return createFailed
}

// reconcileBucketWebsite reports whether a website configuration is present.
// GetBucketWebsite does not return the index and error documents in a form that
// can be compared, so presence is the only available signal.
func reconcileBucketWebsite(service ObjectStorageService, vpcId, s3ServiceId, bucketName string) createOutcome {
	response := service.GetBucketWebsite(vpcId, s3ServiceId, bucketName)
	if response == nil || !response.Status {
		return createFailed
	}
	return createAdopted
}

// reconcileSubUser adopts a sub-user only when the stored role matches, so a
// pre-existing user with different permissions is reported as a conflict.
func reconcileSubUser(service ObjectStorageService, vpcId, s3ServiceId, subUserId, wantRole string) createOutcome {
	detail := service.DetailSubUser(vpcId, s3ServiceId, subUserId)
	if detail == nil || detail.UserID != subUserId {
		return createFailed
	}
	if detail.Role != wantRole {
		return createConflict
	}
	return createAdopted
}

// maxRulesPerPage matches the page size the read functions already use when
// listing a bucket's rules.
const maxRulesPerPage = 999999

func equalStrings(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}

// equalJSON compares two documents by structure rather than by text.
func equalJSON(a, b string) bool {
	var left, right interface{}
	if err := json.Unmarshal([]byte(a), &left); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &right); err != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}
