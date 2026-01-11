package datahub

// GraphQL queries for DataHub API.
const (
	// GetDatasetQuery retrieves metadata for a dataset (table).
	getDatasetQuery = `
query getDataset($urn: String!) {
  dataset(urn: $urn) {
    urn
    properties {
      name
      description
    }
    ownership {
      owners {
        owner {
          ... on CorpUser {
            urn
            username
            properties {
              displayName
              email
            }
          }
          ... on CorpGroup {
            urn
            name
            properties {
              displayName
              email
            }
          }
        }
        ownershipType {
          urn
          info {
            name
          }
        }
      }
    }
    tags {
      tags {
        tag {
          urn
          properties {
            name
            description
          }
        }
      }
    }
    glossaryTerms {
      terms {
        term {
          urn
          properties {
            name
            definition
          }
        }
      }
    }
    domain {
      domain {
        urn
        properties {
          name
          description
        }
      }
    }
    deprecation {
      deprecated
      note
      decommissionTime
    }
  }
}
`

	// GetSchemaQuery retrieves schema metadata including field descriptions.
	getSchemaQuery = `
query getSchema($urn: String!) {
  dataset(urn: $urn) {
    schemaMetadata {
      fields {
        fieldPath
        description
        tags {
          tags {
            tag {
              urn
              properties {
                name
              }
            }
          }
        }
        glossaryTerms {
          terms {
            term {
              urn
              properties {
                name
                definition
              }
            }
          }
        }
      }
    }
  }
}
`

	// GetGlossaryTermQuery retrieves a glossary term by URN.
	getGlossaryTermQuery = `
query getGlossaryTerm($urn: String!) {
  glossaryTerm(urn: $urn) {
    urn
    properties {
      name
      definition
      termSource
    }
    relatedTerms {
      terms {
        term {
          urn
          properties {
            name
          }
        }
        relationshipType
      }
    }
  }
}
`

	// GetLineageQuery retrieves lineage information.
	getLineageQuery = `
query getLineage($urn: String!, $direction: LineageDirection!, $depth: Int) {
  searchAcrossLineage(
    input: {
      urn: $urn
      direction: $direction
      count: 100
    }
  ) {
    searchResults {
      degree
      entity {
        urn
        type
        ... on Dataset {
          name
          properties {
            name
          }
          platform {
            name
          }
        }
      }
    }
  }
}
`

	// SearchDatasetsQuery searches for datasets.
	searchDatasetsQuery = `
query searchDatasets($query: String!, $count: Int!, $filters: [FacetFilterInput!]) {
  search(
    input: {
      type: DATASET
      query: $query
      start: 0
      count: $count
      filters: $filters
    }
  ) {
    searchResults {
      entity {
        urn
        ... on Dataset {
          name
          properties {
            name
            qualifiedName
          }
          platform {
            name
          }
        }
      }
    }
  }
}
`
)
