%{

package assert

%}

%union {
    value Value
    op string
}


%type <value> statement expr

%token <value> VALUE
%token AND
%token OR
%token NOT
%token LB
%token RB
%token E
%token NE
%token RE
%token NRE
%token LT
%token GT
%token LTE
%token GTE
%token EOF

%left EOF
%left OR
%left AND
%left E NE LT GT LTE GTE RE NRE MATCH
%left '+' '-'
%left '*' '/' '%'
%left LB RB
%right NOT
%%

statement: expr EOF
    {
        yylex.(*Assert).answer = $1
        $$ = $1
        return 0
    }

expr: LB expr RB
    {
        $$ = $2
    }
    | NOT expr 
    {
        $$ = $2.Not()
    }
    | expr AND expr 
    {
        $$ = $1.And($3)
    }
    | expr OR expr 
    {
        $$ = $1.Or($3)
    }
    | expr E expr
    { 
        $$ = $1.E($3)
    }
    | expr RE expr
    {
        $$ = $1.RE($3)
    }
    | expr NRE expr
    {
        $$ = $1.NRE($3)
    }
    | expr NE expr
    { 
        $$ = $1.NE($3) 
    }
    | expr LT expr
    {
        $$ = $1.LT($3)
    }
    | expr GT expr
    { 
        $$ = $1.GT($3)
    }
    | expr LTE expr
    {
        $$ = $1.LTE($3)        
    }
    | expr GTE expr
    { 
        $$ = $1.GTE($3)                
    }
    | expr MATCH expr
    {
        $$ = $1.MATCH($3)
    }
    | expr '+' expr
    { 
        $$ = $1.Add($3) 
    }
	| expr '-' expr
    { 
        $$ = $1.Sub($3)
    }
	| expr '*' expr
    {
        $$ = $1.Multi($3)
    }
	| expr '/' expr
    {
        $$ = $1.Div($3)
    }
	| expr '%' expr
    {
        $$ = $1.Mod($3)
    }
    | '-' expr
    {
        $$ = NewValue("", 0).Sub($2)
    }
    | VALUE
    { 
        $$ = $1 
    }
    ;
%%