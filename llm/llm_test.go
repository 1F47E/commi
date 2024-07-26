package llm

import (
	"commi/commit"
	"commi/xmlparser"
	"testing"
)

func TestParseXMLCommit(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		want    *commit.Commit
		wantErr bool
	}{
		{
			name: "Valid XML with tags inside change",
			xml: `
<commit>
	<title>Update commit message structure</title>
	<changes>
		<change>Add &lt;commit&gt;, &lt;title&gt;, &lt;changes&gt;, and &lt;summary&gt; tags</change>
		<change>Remove instruction for blank line after title</change>
	</changes>
	<summary>Restructure commit message format</summary>
</commit>`,
			want: &commit.Commit{
				Title: "Update commit message structure",
				Message: "Add <commit>, <title>, <changes>, and <summary> tags\n" +
					"Remove instruction for blank line after title\n\n" +
					"Restructure commit message format",
			},
			wantErr: false,
		},
		{
			name: "Invalid XML structure",
			xml: `
<commit>
	<title>Invalid XML</title>
	<changes>
		<change>This XML is invalid</change>
	<summary>Missing closing tag for changes</summary>
</commit>`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "Empty changes",
			xml: `
<commit>
	<title>Empty changes</title>
	<changes></changes>
	<summary>No changes made</summary>
</commit>`,
			want: &commit.Commit{
				Title:   "Empty changes",
				Message: "No changes made",
			},
			wantErr: false,
		},
		{
			name: "Multiple changes with nested tags",
			xml: `
<commit>
	<title>Complex commit</title>
	<changes>
		<change>Add support for &lt;nested&gt; XML tags</change>
		<change>Implement &lt;feature1&gt; and &lt;feature2&gt;</change>
		<change>Fix &lt;bug&gt; in &lt;module&gt;</change>
	</changes>
	<summary>Enhance XML handling capabilities</summary>
</commit>`,
			want: &commit.Commit{
				Title: "Complex commit",
				Message: "Add support for <nested> XML tags\n" +
					"Implement <feature1> and <feature2>\n" +
					"Fix <bug> in <module>\n\n" +
					"Enhance XML handling capabilities",
			},
			wantErr: false,
		},
		{
			name: "Invalid XML with malformed nested tags",
			xml: `
<commit>
	<title>Invalid nested tags</title>
	<changes>
		<change>Add support for &lt;nested&gt; XML<title>, <bad> tags</change>
	</changes>
	<summary>This XML is not well-formed</summary>
</commit>`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := xmlparser.ParseXMLCommit(tt.xml)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseXMLCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Title != tt.want.Title {
				t.Errorf("parseXMLCommit() got Title = %v, want %v", got.Title, tt.want.Title)
			}
			if got.Message != tt.want.Message {
				t.Errorf("parseXMLCommit() got Message = %v, want %v", got.Message, tt.want.Message)
			}
		})
	}
}
