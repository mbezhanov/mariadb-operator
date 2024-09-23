package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("SqlJob webhook", func() {
	Context("When creating a SqlJob", func() {
		objMeta := metav1.ObjectMeta{
			Name:      "sqljob-create-webhook",
			Namespace: testNamespace,
		}
		DescribeTable(
			"Should validate",
			func(s *SqlJob, wantErr bool) {
				_ = k8sClient.Delete(testCtx, s)
				err := k8sClient.Create(testCtx, s)
				if wantErr {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry(
				"No SQL",
				&SqlJob{
					ObjectMeta: objMeta,
					Spec: SqlJobSpec{
						MariaDBRef: MariaDBRef{
							ObjectReference: ObjectReference{
								Name: "foo",
							},
						},
						Username: "foo",
						PasswordSecretKeyRef: SecretKeySelector{
							LocalObjectReference: LocalObjectReference{
								Name: "foo",
							},
							Key: "foo",
						},
					},
				},
				true,
			),
			Entry(
				"Invalid schedule",
				&SqlJob{
					ObjectMeta: objMeta,
					Spec: SqlJobSpec{
						MariaDBRef: MariaDBRef{
							ObjectReference: ObjectReference{
								Name: "foo",
							},
						},
						Schedule: &Schedule{
							Cron: "foo",
						},
						Username: "foo",
						PasswordSecretKeyRef: SecretKeySelector{
							LocalObjectReference: LocalObjectReference{
								Name: "foo",
							},
							Key: "foo",
						},
					},
				},
				true,
			),
			Entry(
				"Invalid history limits",
				&SqlJob{
					ObjectMeta: objMeta,
					Spec: SqlJobSpec{
						MariaDBRef: MariaDBRef{
							ObjectReference: ObjectReference{
								Name: "foo",
							},
						},
						Schedule: &Schedule{
							Cron: "foo",
						},
						Username: "foo",
						PasswordSecretKeyRef: SecretKeySelector{
							LocalObjectReference: LocalObjectReference{
								Name: "foo",
							},
							Key: "foo",
						},
						Sql: func() *string { s := "foo"; return &s }(),
						CronJobTemplate: CronJobTemplate{
							SuccessfulJobsHistoryLimit: ptr.To[int32](-5),
							FailedJobsHistoryLimit:     ptr.To[int32](-5),
						},
					},
				},
				true,
			),
			Entry(
				"Valid",
				&SqlJob{
					ObjectMeta: objMeta,
					Spec: SqlJobSpec{
						MariaDBRef: MariaDBRef{
							ObjectReference: ObjectReference{
								Name: "foo",
							},
						},
						Username: "foo",
						PasswordSecretKeyRef: SecretKeySelector{
							LocalObjectReference: LocalObjectReference{
								Name: "foo",
							},
							Key: "foo",
						},
						Sql: func() *string { s := "foo"; return &s }(),
					},
				},
				false,
			),
			Entry(
				"Valid with schedule",
				&SqlJob{
					ObjectMeta: objMeta,
					Spec: SqlJobSpec{
						MariaDBRef: MariaDBRef{
							ObjectReference: ObjectReference{
								Name: "foo",
							},
						},
						Schedule: &Schedule{
							Cron: "*/1 * * * *",
						},
						Username: "foo",
						PasswordSecretKeyRef: SecretKeySelector{
							LocalObjectReference: LocalObjectReference{
								Name: "foo",
							},
							Key: "foo",
						},
						Sql: func() *string { s := "foo"; return &s }(),
					},
				},
				false,
			),
			Entry(
				"Valid with schedule and history limits",
				&SqlJob{
					ObjectMeta: objMeta,
					Spec: SqlJobSpec{
						MariaDBRef: MariaDBRef{
							ObjectReference: ObjectReference{
								Name: "foo",
							},
						},
						Schedule: &Schedule{
							Cron: "foo",
						},
						Username: "foo",
						PasswordSecretKeyRef: SecretKeySelector{
							LocalObjectReference: LocalObjectReference{
								Name: "foo",
							},
							Key: "foo",
						},
						Sql: func() *string { s := "foo"; return &s }(),
						CronJobTemplate: CronJobTemplate{
							SuccessfulJobsHistoryLimit: ptr.To[int32](5),
							FailedJobsHistoryLimit:     ptr.To[int32](5),
						},
					},
				},
				true,
			),
		)
	})

	Context("When updating a SqlJob", Ordered, func() {
		key := types.NamespacedName{
			Name:      "sqljob-update-webhook",
			Namespace: testNamespace,
		}
		BeforeAll(func() {
			sqlJob := SqlJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: SqlJobSpec{
					DependsOn: []LocalObjectReference{
						{
							Name: "sqljob-webhook",
						},
					},
					MariaDBRef: MariaDBRef{
						ObjectReference: ObjectReference{
							Name: "mariadb-webhook",
						},
						WaitForIt: true,
					},
					Username: "test",
					PasswordSecretKeyRef: SecretKeySelector{
						LocalObjectReference: LocalObjectReference{
							Name: "test",
						},
						Key: "test",
					},
					Database: func() *string { d := "test"; return &d }(),
					Sql: func() *string {
						sql := `CREATE TABLE IF NOT EXISTS users (
							id bigint PRIMARY KEY AUTO_INCREMENT,
							username varchar(255) NOT NULL,
							email varchar(255) NOT NULL,
							UNIQUE KEY name__unique_idx (username),
							UNIQUE KEY email__unique_idx (email)
						);`
						return &sql
					}(),
				},
			}
			Expect(k8sClient.Create(testCtx, &sqlJob)).To(Succeed())
		})
		DescribeTable(
			"Should validate",
			func(patchFn func(job *SqlJob), wantErr bool) {
				var sqlJob SqlJob
				Expect(k8sClient.Get(testCtx, key, &sqlJob)).To(Succeed())

				patch := client.MergeFrom(sqlJob.DeepCopy())
				patchFn(&sqlJob)

				err := k8sClient.Patch(testCtx, &sqlJob, patch)
				if wantErr {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry(
				"Updating BackoffLimit",
				func(job *SqlJob) {
					job.Spec.BackoffLimit = 20
				},
				false,
			),
			Entry(
				"Updating RestartPolicy",
				func(job *SqlJob) {
					job.Spec.RestartPolicy = corev1.RestartPolicyNever
				},
				true,
			),
			Entry(
				"Updating Resources",
				func(job *SqlJob) {
					job.Spec.Resources = &ResourceRequirements{
						Requests: corev1.ResourceList{
							"cpu": resource.MustParse("200m"),
						},
					}
				},
				false,
			),
			Entry(
				"Updating MariaDBRef",
				func(job *SqlJob) {
					job.Spec.MariaDBRef.Name = "another-mariadb"
				},
				true,
			),
			Entry(
				"Updating Username",
				func(job *SqlJob) {
					job.Spec.Username = "foo"
				},
				true,
			),
			Entry(
				"Updating PasswordSecretKeyRef",
				func(job *SqlJob) {
					job.Spec.PasswordSecretKeyRef.Name = "foo"
				},
				true,
			),
			Entry(
				"Updating Database",
				func(job *SqlJob) {
					job.Spec.Database = func() *string { d := "foo"; return &d }()
				},
				true,
			),
			Entry(
				"Updating DependsOn",
				func(job *SqlJob) {
					job.Spec.DependsOn = nil
				},
				true,
			),
			Entry(
				"Updating Sql",
				func(job *SqlJob) {
					job.Spec.Sql = func() *string { d := "foo"; return &d }()
				},
				true,
			),
			Entry(
				"Updating SqlConfigMapKeyRef",
				func(job *SqlJob) {
					job.Spec.SqlConfigMapKeyRef = &ConfigMapKeySelector{
						LocalObjectReference: LocalObjectReference{
							Name: "foo",
						},
					}
				},
				false,
			),
			Entry(
				"Updating Schedule",
				func(job *SqlJob) {
					job.Spec.Schedule = &Schedule{
						Cron:    "*/1 * * * *",
						Suspend: false,
					}
				},
				false,
			),
			Entry(
				"Updating with wrong Schedule",
				func(job *SqlJob) {
					job.Spec.Schedule = &Schedule{
						Cron:    "foo",
						Suspend: false,
					}
				},
				true,
			),
			Entry(
				"Updating SuccessfulJobsHistoryLimit",
				func(job *SqlJob) {
					job.Spec.SuccessfulJobsHistoryLimit = ptr.To[int32](5)
				},
				false,
			),
			Entry(
				"Updating with wrong SuccessfulJobsHistoryLimit",
				func(job *SqlJob) {
					job.Spec.SuccessfulJobsHistoryLimit = ptr.To[int32](-5)
				},
				true,
			),
			Entry(
				"Updating FailedJobsHistoryLimit",
				func(job *SqlJob) {
					job.Spec.FailedJobsHistoryLimit = ptr.To[int32](5)
				},
				false,
			),
			Entry(
				"Updating with wrong FailedJobsHistoryLimit",
				func(job *SqlJob) {
					job.Spec.FailedJobsHistoryLimit = ptr.To[int32](-5)
				},
				true,
			),
			Entry(
				"Removing SQL",
				func(job *SqlJob) {
					job.Spec.Sql = nil
					job.Spec.SqlConfigMapKeyRef = nil
				},
				true,
			),
		)
	})
})
