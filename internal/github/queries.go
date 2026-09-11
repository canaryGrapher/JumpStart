package github

// GraphQL documents. They live together so the shape of what JumpStart
// asks GitHub for is readable in one place.

const queryViewer = `
query { viewer { login name avatarUrl bio company location websiteUrl url } }`

// fieldsFragment pulls every field definition on a board, including the
// options of single-selects and the cycles of iteration fields, so the
// UI can render a native editor for each one.
const fieldsFragment = `
fields(first: 50) {
  nodes {
    ... on ProjectV2FieldCommon { id name dataType }
    ... on ProjectV2SingleSelectField {
      options { id name color description }
    }
    ... on ProjectV2IterationField {
      configuration {
        iterations { id title startDate duration }
        completedIterations { id title startDate duration }
      }
    }
  }
}`

// queryOwnerProjects lists the boards owned by a login. The same
// document serves users and organizations by asking for both.
const queryOwnerProjects = `
query($login: String!, $cursor: String) {
  user(login: $login) {
    projectsV2(first: 50, after: $cursor, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes { id number title url shortDescription closed public updatedAt }
    }
  }
  organization(login: $login) {
    projectsV2(first: 50, after: $cursor, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes { id number title url shortDescription closed public updatedAt }
    }
  }
}`

// queryViewerProjects lists boards the signed-in account owns, used as
// the default suggestion when linking a project.
const queryViewerProjects = `
query($cursor: String) {
  viewer {
    login
    projectsV2(first: 50, after: $cursor, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes { id number title url shortDescription closed public updatedAt }
    }
  }
}`

// queryProjectFields fetches one board's metadata and field definitions.
var queryProjectFields = `
query($id: ID!) {
  node(id: $id) {
    ... on ProjectV2 {
      id number title url shortDescription closed public updatedAt
      owner { __typename ... on User { login } ... on Organization { login } }
      ` + fieldsFragment + `
    }
  }
}`

// queryProjectItems walks a board's rows. Every built-in Projects v2
// column is requested here: assignees, labels, milestone, repository,
// reviewers, parent issue, sub-issue progress, linked pull requests and
// issue type, alongside the custom field values.
const queryProjectItems = `
query($id: ID!, $cursor: String) {
  node(id: $id) {
    ... on ProjectV2 {
      items(first: 50, after: $cursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          isArchived
          updatedAt
          type
          content {
            ... on DraftIssue { id title body updatedAt }
            ... on Issue {
              id number title body url state updatedAt
              issueType { name }
              repository { nameWithOwner }
              milestone { title }
              parent { title }
              assignees(first: 20) { nodes { login } }
              labels(first: 20) { nodes { name } }
            }
            ... on PullRequest {
              id number title body url state updatedAt
              repository { nameWithOwner }
              milestone { title }
              assignees(first: 20) { nodes { login } }
              labels(first: 20) { nodes { name } }
              reviewRequests(first: 20) {
                nodes { requestedReviewer { ... on User { login } ... on Team { name } } }
              }
            }
          }
          fieldValues(first: 50) {
            nodes {
              ... on ProjectV2ItemFieldTextValue {
                text field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldNumberValue {
                number field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldDateValue {
                date field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldSingleSelectValue {
                optionId name field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldIterationValue {
                iterationId title field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldMilestoneValue {
                milestone { title } field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldRepositoryValue {
                repository { nameWithOwner } field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldLabelValue {
                labels(first: 20) { nodes { name } }
                field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldUserValue {
                users(first: 20) { nodes { login } }
                field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldPullRequestValue {
                pullRequests(first: 10) { nodes { url } }
                field { ... on ProjectV2FieldCommon { id name dataType } }
              }
              ... on ProjectV2ItemFieldReviewerValue {
                reviewers(first: 20) {
                  nodes { ... on User { login } ... on Team { name } }
                }
                field { ... on ProjectV2FieldCommon { id name dataType } }
              }
            }
          }
        }
      }
    }
  }
}`

const mutationAddDraft = `
mutation($projectId: ID!, $title: String!, $body: String) {
  addProjectV2DraftIssue(input: {projectId: $projectId, title: $title, body: $body}) {
    projectItem { id }
  }
}`

const mutationAddItem = `
mutation($projectId: ID!, $contentId: ID!) {
  addProjectV2ItemById(input: {projectId: $projectId, contentId: $contentId}) {
    item { id }
  }
}`

const mutationDeleteItem = `
mutation($projectId: ID!, $itemId: ID!) {
  deleteProjectV2Item(input: {projectId: $projectId, itemId: $itemId}) { deletedItemId }
}`

const mutationUpdateDraft = `
mutation($draftId: ID!, $title: String, $body: String) {
  updateProjectV2DraftIssue(input: {draftIssueId: $draftId, title: $title, body: $body}) {
    draftIssue { id }
  }
}`

const mutationUpdateIssue = `
mutation($issueId: ID!, $title: String, $body: String) {
  updateIssue(input: {id: $issueId, title: $title, body: $body}) { issue { id } }
}`

const mutationSetIssueAssignees = `
mutation($issueId: ID!, $assigneeIds: [ID!]!) {
  updateIssue(input: {id: $issueId, assigneeIds: $assigneeIds}) { issue { id } }
}`

const queryAssignableUsers = `
query($owner: String!, $name: String!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    assignableUsers(first: 100, after: $cursor) {
      pageInfo { hasNextPage endCursor }
      nodes { id login name avatarUrl }
    }
  }
}`

const queryUserByLogin = `
query($login: String!) {
  user(login: $login) { id login name avatarUrl }
}`

const mutationSetFieldValue = `
mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: ProjectV2FieldValue!) {
  updateProjectV2ItemFieldValue(
    input: {projectId: $projectId, itemId: $itemId, fieldId: $fieldId, value: $value}
  ) { projectV2Item { id } }
}`

const mutationClearFieldValue = `
mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!) {
  clearProjectV2ItemFieldValue(
    input: {projectId: $projectId, itemId: $itemId, fieldId: $fieldId}
  ) { projectV2Item { id } }
}`

const mutationCreateIssue = `
mutation($repoId: ID!, $title: String!, $body: String) {
  createIssue(input: {repositoryId: $repoId, title: $title, body: $body}) {
    issue { id number url }
  }
}`

const queryRepositories = `
query($cursor: String) {
  viewer {
    repositories(
      first: 50, after: $cursor, affiliations: [OWNER, COLLABORATOR, ORGANIZATION_MEMBER]
      orderBy: {field: PUSHED_AT, direction: DESC}
    ) {
      pageInfo { hasNextPage endCursor }
      nodes { id name url owner { login } nameWithOwner }
    }
  }
}`

// queryRepoIssues lists a repository's issues, most recently updated
// first, for the linked-repository panel in the task tracker.
const queryRepoIssues = `
query($owner: String!, $name: String!, $states: [IssueState!], $limit: Int!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    issues(first: $limit, after: $cursor, states: $states, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id number title url state createdAt updatedAt
        author { login }
        comments { totalCount }
        labels(first: 10) { nodes { name } }
      }
    }
  }
}`

// queryRepoPullRequests lists a repository's pull requests the same way.
const queryRepoPullRequests = `
query($owner: String!, $name: String!, $states: [PullRequestState!], $limit: Int!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequests(first: $limit, after: $cursor, states: $states, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id number title url state isDraft createdAt updatedAt
        baseRefName headRefName
        author { login }
        comments { totalCount }
      }
    }
  }
}`

// queryRepoBranches lists a repository's live branches on GitHub —
// the authoritative answer to "does this branch exist on the remote",
// independent of whatever the local clone last fetched.
const queryRepoBranches = `
query($owner: String!, $name: String!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    refs(refPrefix: "refs/heads/", first: 100, after: $cursor) {
      pageInfo { hasNextPage endCursor }
      nodes {
        name
        target {
          oid
          ... on Commit { committedDate }
        }
      }
    }
  }
}`

const mutationCreatePullRequest = `
mutation($repoId: ID!, $base: String!, $head: String!, $title: String!, $body: String, $draft: Boolean) {
  createPullRequest(
    input: {repositoryId: $repoId, baseRefName: $base, headRefName: $head, title: $title, body: $body, draft: $draft}
  ) {
    pullRequest { id number url }
  }
}`

// queryViewerOrgs lists the organizations the signed-in account belongs
// to, so the "connect GitHub" flow can offer them alongside the personal
// account as a place to create or pick a repository.
const queryViewerOrgs = `
query($cursor: String) {
  viewer {
    id
    login
    avatarUrl
    organizations(first: 100, after: $cursor) {
      pageInfo { hasNextPage endCursor }
      nodes { id login avatarUrl name }
    }
  }
}`

// mutationCreateProjectV2 creates a new Projects v2 board under an
// owner's node id. GitHub seeds it with its own default fields
// (including a Status single-select with Todo/In Progress/Done) — there
// is no way to request one of GitHub's own template layouts through the
// public API, so ApplyStatusPreset re-labels the Status field afterward
// instead.
const mutationCreateProjectV2 = `
mutation($ownerId: ID!, $title: String!) {
  createProjectV2(input: {ownerId: $ownerId, title: $title}) {
    projectV2 {
      id number title url shortDescription closed public updatedAt
      owner { __typename ... on User { login } ... on Organization { login } }
    }
  }
}`

// mutationCreateIterationField creates a Projects v2 ITERATION field with
// an initial set of cycles. Used when a CSV import invents sprint names
// and the linked board has no iteration field yet.
const mutationCreateIterationField = `
mutation($projectId: ID!, $name: String!, $configuration: ProjectV2IterationFieldConfigurationInput!) {
  createProjectV2Field(input: {
    projectId: $projectId
    dataType: ITERATION
    name: $name
    iterationConfiguration: $configuration
  }) {
    projectV2Field {
      ... on ProjectV2IterationField {
        id
        name
        configuration {
          iterations { id title startDate duration }
        }
      }
    }
  }
}`

// mutationUpdateIterationField replaces the iteration configuration on an
// existing ITERATION field. GitHub overwrites the full list (ids are
// regenerated), so callers must pass every cycle they want to keep.
const mutationUpdateIterationField = `
mutation($fieldId: ID!, $configuration: ProjectV2IterationFieldConfigurationInput!) {
  updateProjectV2Field(input: {
    fieldId: $fieldId
    iterationConfiguration: $configuration
  }) {
    projectV2Field {
      ... on ProjectV2IterationField {
        id
        name
        configuration {
          iterations { id title startDate duration }
          completedIterations { id title startDate duration }
        }
      }
    }
  }
}`

// mutationUpdateSingleSelectField replaces the full set of options on a
// single-select field (e.g. a board's Status column), in the given
// order.
const mutationUpdateSingleSelectField = `
mutation($fieldId: ID!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {
  updateProjectV2Field(input: {fieldId: $fieldId, singleSelectOptions: $options}) {
    projectV2Field {
      ... on ProjectV2SingleSelectField { id name options { id name } }
    }
  }
}`

// mutationLinkProjectV2ToRepository links a Projects v2 board to a
// repository. This is separate from JumpStart's own sync link
// (GitHubSync.Repo/RepoID, which only tells the local poller what to
// read): GitHub's repository "Projects" tab only lists boards that have
// been linked this way, so without it a freshly created board is fully
// functional for syncing but invisible from the repo page on github.com.
const mutationLinkProjectV2ToRepository = `
mutation($projectId: ID!, $repositoryId: ID!) {
  linkProjectV2ToRepository(input: {projectId: $projectId, repositoryId: $repositoryId}) {
    repository { id }
  }
}`

const queryRepoByName = `
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) { id name url nameWithOwner owner { login } }
}`
