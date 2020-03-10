//! Wrapper types for the Gitlab API
//!
//! We would use the [gitlab](https://crates.io/crate/gitlab) crate but this is a large additional
//! dependency where we in fact only want a couple of the keys out of the JSON object.

use serde::Deserialize;

#[derive(Deserialize, Debug)]
#[serde(tag = "object_kind")]
pub(crate) enum Event {
    #[serde(rename = "pipeline")]
    Pipeline(Pipeline),
    #[serde(rename = "push")]
    Push,
    #[serde(rename = "tag_push")]
    TagPush,
    #[serde(rename = "build")]
    Build,
    #[serde(rename = "issue")]
    Issue,
    #[serde(rename = "note")]
    Note,
    #[serde(rename = "merge_request")]
    MergeRequest,
    #[serde(rename = "wiki_page")]
    WikiPage,
}

#[derive(Deserialize, Debug)]
pub(crate) struct Pipeline {
    pub(crate) builds: Vec<Build>,
    pub(crate) object_attributes: ObjectAttributes,
}

impl Pipeline {
    pub(crate) fn should_rerun_pipeline(&self) -> bool {
        self.is_master() && self.has_builds() && self.all_builds_passed_or_skipped()
    }

    fn is_master(&self) -> bool {
        self.object_attributes.object_ref == "master"
    }

    fn has_builds(&self) -> bool {
        !self.builds.is_empty()
    }

    fn all_builds_passed_or_skipped(&self) -> bool {
        self.builds
            .iter()
            .all(|b| b.status == Status::Success || b.status == Status::Skipped)
    }
}

#[derive(Deserialize, Debug)]
pub(crate) struct ObjectAttributes {
    #[serde(rename = "ref")]
    pub(crate) object_ref: String,
}

#[derive(Deserialize, Debug)]
pub(crate) struct Build {
    pub(crate) status: Status,
}

#[derive(Deserialize, Debug, PartialEq, Eq, Clone, Copy)]
pub(crate) enum Status {
    #[serde(rename = "skipped")]
    Skipped,
    #[serde(rename = "success")]
    Success,
    #[serde(rename = "created")]
    Created,
    #[serde(rename = "failed")]
    Failed,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_example_webhook_event() {
        let res: Event = serde_json::from_str(WEBHOOK_EVENT).unwrap();
        match res {
            Event::Pipeline(pipeline) => {
                assert_eq!(pipeline.object_attributes.object_ref, "master");

                let build_statuses: Vec<_> = pipeline.builds.iter().map(|b| b.status).collect();
                assert_eq!(
                    build_statuses,
                    vec![
                        Status::Skipped,
                        Status::Success,
                        Status::Success,
                        Status::Success,
                        Status::Created
                    ]
                );
            }
            _ => unreachable!("wrong event type"),
        }
    }

    #[test]
    fn should_rerun_pipeline() {
        let event = Pipeline {
            builds: vec![
                Build {
                    status: Status::Success,
                },
                Build {
                    status: Status::Skipped,
                },
            ],
            object_attributes: ObjectAttributes {
                object_ref: "master".to_string(),
            },
        };

        assert!(
            event.should_rerun_pipeline(),
            "should_rerun_pipeline should be true, found false"
        );
    }

    #[test]
    fn should_rerun_pipeline_non_master() {
        let event = Pipeline {
            builds: vec![
                Build {
                    status: Status::Success,
                },
                Build {
                    status: Status::Skipped,
                },
                Build {
                    status: Status::Created,
                },
            ],
            object_attributes: ObjectAttributes {
                object_ref: "not-master".to_string(),
            },
        };

        assert!(
            !event.should_rerun_pipeline(),
            "should_rerun_pipeline should not run but did"
        );
    }

    #[test]
    fn should_rerun_pipeline_failed_build() {
        let event = Pipeline {
            builds: vec![
                Build {
                    status: Status::Success,
                },
                Build {
                    status: Status::Failed,
                },
                Build {
                    status: Status::Created,
                },
            ],
            object_attributes: ObjectAttributes {
                object_ref: "master".to_string(),
            },
        };

        assert!(
            !event.should_rerun_pipeline(),
            "should_rerun_pipeline should not run but did"
        );
    }
    static WEBHOOK_EVENT: &str = r#"
    {
   "object_kind": "pipeline",
   "object_attributes":{
      "id": 31,
      "ref": "master",
      "tag": false,
      "sha": "bcbb5ec396a2c0f828686f14fac9b80b780504f2",
      "before_sha": "bcbb5ec396a2c0f828686f14fac9b80b780504f2",
      "source": "merge_request_event",
      "status": "success",
      "stages":[
         "build",
         "test",
         "deploy"
      ],
      "created_at": "2016-08-12 15:23:28 UTC",
      "finished_at": "2016-08-12 15:26:29 UTC",
      "duration": 63,
      "variables": [
        {
          "key": "NESTOR_PROD_ENVIRONMENT",
          "value": "us-west-1"
        }
      ]
   },
    "merge_request": {
      "id": 1,
      "iid": 1,
      "title": "Test",
      "source_branch": "test",
      "source_project_id": 1,
      "target_branch": "master",
      "target_project_id": 1,
      "state": "opened",
      "merge_status": "can_be_merged",
      "url": "http://192.168.64.1:3005/gitlab-org/gitlab-test/merge_requests/1"
   },
   "user":{
      "name": "Administrator",
      "username": "root",
      "avatar_url": "http://www.gravatar.com/avatar/e32bd13e2add097461cb96824b7a829c?s=80\u0026d=identicon",
      "email": "user_email@gitlab.com"
   },
   "project":{
      "id": 1,
      "name": "Gitlab Test",
      "description": "Atque in sunt eos similique dolores voluptatem.",
      "web_url": "http://192.168.64.1:3005/gitlab-org/gitlab-test",
      "avatar_url": null,
      "git_ssh_url": "git@192.168.64.1:gitlab-org/gitlab-test.git",
      "git_http_url": "http://192.168.64.1:3005/gitlab-org/gitlab-test.git",
      "namespace": "Gitlab Org",
      "visibility_level": 20,
      "path_with_namespace": "gitlab-org/gitlab-test",
      "default_branch": "master"
   },
   "commit":{
      "id": "bcbb5ec396a2c0f828686f14fac9b80b780504f2",
      "message": "test\n",
      "timestamp": "2016-08-12T17:23:21+02:00",
      "url": "http://example.com/gitlab-org/gitlab-test/commit/bcbb5ec396a2c0f828686f14fac9b80b780504f2",
      "author":{
         "name": "User",
         "email": "user@gitlab.com"
      }
   },
   "builds":[
      {
         "id": 380,
         "stage": "deploy",
         "name": "production",
         "status": "skipped",
         "created_at": "2016-08-12 15:23:28 UTC",
         "started_at": null,
         "finished_at": null,
         "when": "manual",
         "manual": true,
         "allow_failure": false,
         "user":{
            "name": "Administrator",
            "username": "root",
            "avatar_url": "http://www.gravatar.com/avatar/e32bd13e2add097461cb96824b7a829c?s=80\u0026d=identicon"
         },
         "runner": null,
         "artifacts_file":{
            "filename": null,
            "size": null
         }
      },
      {
         "id": 377,
         "stage": "test",
         "name": "test-image",
         "status": "success",
         "created_at": "2016-08-12 15:23:28 UTC",
         "started_at": "2016-08-12 15:26:12 UTC",
         "finished_at": null,
         "when": "on_success",
         "manual": false,
         "allow_failure": false,
         "user":{
            "name": "Administrator",
            "username": "root",
            "avatar_url": "http://www.gravatar.com/avatar/e32bd13e2add097461cb96824b7a829c?s=80\u0026d=identicon"
         },
         "runner": {
            "id":380987,
            "description":"shared-runners-manager-6.gitlab.com",
            "active":true,
            "is_shared":true
         },
         "artifacts_file":{
            "filename": null,
            "size": null
         }
      },
      {
         "id": 378,
         "stage": "test",
         "name": "test-build",
         "status": "success",
         "created_at": "2016-08-12 15:23:28 UTC",
         "started_at": "2016-08-12 15:26:12 UTC",
         "finished_at": "2016-08-12 15:26:29 UTC",
         "when": "on_success",
         "manual": false,
         "allow_failure": false,
         "user":{
            "name": "Administrator",
            "username": "root",
            "avatar_url": "http://www.gravatar.com/avatar/e32bd13e2add097461cb96824b7a829c?s=80\u0026d=identicon"
         },
         "runner": {
            "id":380987,
            "description":"shared-runners-manager-6.gitlab.com",
            "active":true,
            "is_shared":true
         },
         "artifacts_file":{
            "filename": null,
            "size": null
         }
      },
      {
         "id": 376,
         "stage": "build",
         "name": "build-image",
         "status": "success",
         "created_at": "2016-08-12 15:23:28 UTC",
         "started_at": "2016-08-12 15:24:56 UTC",
         "finished_at": "2016-08-12 15:25:26 UTC",
         "when": "on_success",
         "manual": false,
         "allow_failure": false,
         "user":{
            "name": "Administrator",
            "username": "root",
            "avatar_url": "http://www.gravatar.com/avatar/e32bd13e2add097461cb96824b7a829c?s=80\u0026d=identicon"
         },
         "runner": {
            "id":380987,
            "description":"shared-runners-manager-6.gitlab.com",
            "active":true,
            "is_shared":true
         },
         "artifacts_file":{
            "filename": null,
            "size": null
         }
      },
      {
         "id": 379,
         "stage": "deploy",
         "name": "staging",
         "status": "created",
         "created_at": "2016-08-12 15:23:28 UTC",
         "started_at": null,
         "finished_at": null,
         "when": "on_success",
         "manual": false,
         "allow_failure": false,
         "user":{
            "name": "Administrator",
            "username": "root",
            "avatar_url": "http://www.gravatar.com/avatar/e32bd13e2add097461cb96824b7a829c?s=80\u0026d=identicon"
         },
         "runner": null,
         "artifacts_file":{
            "filename": null,
            "size": null
         }
      }
   ]
}
"#;
}
