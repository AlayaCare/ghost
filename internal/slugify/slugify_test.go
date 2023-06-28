package slugify

import "testing"

func TestMake(t *testing.T) {
	type args struct {
		str         string
		replacerMap []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test Default Replacer Map",
			args: args{
				str:         "Çikolata Soslu Bulut Kek",
				replacerMap: []string{},
			},
			want: "cikolata-soslu-bulut-kek",
		},
		{
			name: "Test Complex Sentence",
			args: args{
				str:         "Çikolata'nın Sosunda /Kendi ?Halinde -- Ne Üdüğü Belirsiz == -- - \" Bulut Kek",
				replacerMap: []string{},
			},
			want: "cikolatanin-sosunda-kendi-halinde-ne-udugu-belirsiz-bulut-kek",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Make(tt.args.str, tt.args.replacerMap...); got != tt.want {
				t.Errorf("Make() = %v, want %v", got, tt.want)
			}
		})
	}
}
