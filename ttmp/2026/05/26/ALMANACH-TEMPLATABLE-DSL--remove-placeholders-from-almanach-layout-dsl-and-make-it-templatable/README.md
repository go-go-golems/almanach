# Remove placeholders from almanach layout DSL and make it templatable

This is the document workspace for ticket ALMANACH-TEMPLATABLE-DSL.

## Structure

- **design/**: Design documents and architecture notes
- **reference/**: Reference documentation and API contracts
- **playbooks/**: Operational playbooks and procedures
- **scripts/**: Utility scripts and automation
- **sources/**: External sources and imported documents
- **various/**: Scratch or meeting notes, working notes
- **archive/**: Optional space for deprecated or reference-only artifacts

## Getting Started

Use docmgr commands to manage this workspace:

- Add documents: `docmgr doc add --ticket ALMANACH-TEMPLATABLE-DSL --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket ALMANACH-TEMPLATABLE-DSL --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket ALMANACH-TEMPLATABLE-DSL --field Status --value review`
