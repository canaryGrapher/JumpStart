package github

// GraphQL documents. They live together so the shape of what JumpStart
// asks GitHub for is readable in one place.

const queryViewer = `
query { viewer { login name avatarUrl } }`

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

const queryRepoByName = `
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) { id name url nameWithOwner owner { login } }
}`
