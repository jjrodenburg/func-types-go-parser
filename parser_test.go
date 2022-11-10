package main

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_splitPrototypeIntoTokens(t *testing.T) {
	type args struct {
		funcPrototype string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "empty prototype",
			args: args{funcPrototype: " "},
			want: nil,
		},
		{
			name: "valid prototype with receiver and type inference",
			args: args{funcPrototype: "inc :: Num a => a -> a"},
			want: []string{"inc", "::", "Num", "a", "=>", "a", "->", "a"},
		},
		{
			name: "valid prototype with no receiver",
			args: args{funcPrototype: "inc :: Num a -> a"},
			want: []string{"inc", "::", "Num", "a", "->", "a"},
		},
		{
			name: "valid prototype with multiple returns",
			args: args{funcPrototype: "inc :: Num a -> (a -> a)"},
			want: []string{"inc", "::", "Num", "a", "->", "(", "a", "->", "a", ")"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitPrototypeIntoTokens(tt.args.funcPrototype); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitPrototypeIntoTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name          string
		funcPrototype string
		want          ast.FuncDecl
		wantErr       bool
	}{
		{
			name:          "valid prototype with one input argument integer and one output argument string",
			funcPrototype: "TestFunc :: Int a -> String b",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "String"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}}}}},
			},
			wantErr: false,
		},
		{
			name:          "invalid prototype with one input argument integer and multiple outputs with no closing block",
			funcPrototype: "TestFunc :: Int a -> String b, c",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "valid prototype with input type inference",
			funcPrototype: "TestFunc :: Int a -> z -> String b",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "String"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
					{Names: []*ast.Ident{{Name: "z"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "valid prototype with input and output type inference",
			funcPrototype: "TestFunc :: Int a -> z -> b",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "Int"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
					{Names: []*ast.Ident{{Name: "z"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "invalid prototype with trailing ->",
			funcPrototype: "TestFunc :: Int a -> z ->",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "invalid prototype with typed argument with no name",
			funcPrototype: "TestFunc :: Int -> z",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "valid prototype with typed argument with inferred typed name looking like type",
			funcPrototype: "TestFunc :: Int a -> Bool",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "Bool"}}, Type: &ast.Ident{Name: "Int"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "valid prototype with no input and one output",
			funcPrototype: "TestFunc :: Float64 a",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Float64"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{},
			},
			wantErr: false,
		},
		{
			name:          "invalid prototype with untyped output argument",
			funcPrototype: "TestFunc :: a",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "invalid prototype with operator token followed by no params",
			funcPrototype: "TestFunc ::",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "valid prototype with no input or output",
			funcPrototype: "TestFunc",
			want: ast.FuncDecl{
				Name: &ast.Ident{Name: "TestFunc"},
			},
			wantErr: false,
		},

		{
			name:          "invalid prototype with incorrect operator token character",
			funcPrototype: "TestFunc :",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "invalid prototype with one input argument integer and two output arguments seperated by a comma",
			funcPrototype: "TestFunc :: Int a -> (String b, Int c)",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "valid prototype with one input and Two outputs",
			funcPrototype: "TestFunc :: Int a -> (String b -> Int c)",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "String"}}, {Names: []*ast.Ident{{Name: "c"}}, Type: &ast.Ident{Name: "Int"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "Invalid prototype with one input and two unwrapped (close) output",
			funcPrototype: "TestFunc :: Int a -> (String b -> Int c",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "Invalid prototype with one input and two unwrapped (start) output",
			funcPrototype: "TestFunc :: Int a -> String b -> Int c)",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "valid prototype with one input and one wrapped output",
			funcPrototype: "TestFunc :: Int a -> (Int b)",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "Int"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "invalid prototype with one input and one type inferred wrapped output with forbidden char in name",
			funcPrototype: "TestFunc :: Int a -> (b)",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "valid prototype with one input and one type inferred wrapped output",
			funcPrototype: "TestFunc :: Int a -> ( b )",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "Int"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "valid prototype with one input and multiple type inferred wrapped output",
			funcPrototype: "TestFunc :: Int a -> ( b -> c )",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "Int"}}, {Names: []*ast.Ident{{Name: "c"}}, Type: &ast.Ident{Name: "Int"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "valid prototype with one input and one type inferred and one typed wrapped output",
			funcPrototype: "TestFunc :: Int a -> ( b -> String c )",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "Int"}}, {Names: []*ast.Ident{{Name: "c"}}, Type: &ast.Ident{Name: "String"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "valid prototype with one input and one typed followed by inferred output",
			funcPrototype: "TestFunc :: Int a -> ( String b -> c )",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "b"}}, Type: &ast.Ident{Name: "String"}}, {Names: []*ast.Ident{{Name: "c"}}, Type: &ast.Ident{Name: "String"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "invalid prototype with arguments after wrapped output block",
			funcPrototype: "TestFunc :: Int a -> ( String b -> c ) -> d",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "invalid prototype with duplicate parameter names",
			funcPrototype: "TestFunc :: Int a -> a",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
		{
			name:          "valid prototype with same parameter names for input and output",
			funcPrototype: "TestFunc :: Int a -> (Int a)",
			want: ast.FuncDecl{
				Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}}}},
				Name: &ast.Ident{Name: "TestFunc"},
				Type: &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "a"}}, Type: &ast.Ident{Name: "Int"}},
				}}},
			},
			wantErr: false,
		},
		{
			name:          "invalid prototype with duplicate multi output parameter names",
			funcPrototype: "TestFunc :: Int a -> (String b -> b)",
			want:          ast.FuncDecl{},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.funcPrototype)
			if tt.wantErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}
