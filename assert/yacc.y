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
        yylex.(*Assert).answer = $1.Boolean()
        $$ = $1
        return 0
    }

expr: LB expr RB
    {
        $$ = $2
    }
    | NOT expr 
    {
        var err int
        $$, err = $2.Not()
        if err != NoError {
            return err
        }
    }
    | expr AND expr 
    {
        var err int
        $$, err = $1.And($3)
        if err != NoError {
            return err
        }
    }
    | expr OR expr 
    {
        var err int
        $$, err = $1.Or($3)
        if err != NoError {
            return err
        }
    }
    | expr E expr
    { 
        var err int
        $$, err = $1.E($3)
        if err != NoError {
            return err
        }
    }
    | expr RE expr
    {
        var err int
        $$, err = $1.RE($3)
        if err != NoError {
            return err
        }
    }
    | expr NRE expr
    {
        var err int
        $$, err = $1.NRE($3)
        if err != NoError {
            return err
        }
    }
    | expr NE expr
    { 
        var err int
        $$, err = $1.NE($3) 
        if err != NoError {
            return err
        }
    }
    | expr LT expr
    {
        var err int
        $$, err = $1.LT($3)
        if err != NoError {
            return err
        }
    }
    | expr GT expr
    { 
        var err int
        $$, err = $1.GT($3)
        if err != NoError {
            return err
        }
    }
    | expr LTE expr
    {
        var err int
        $$, err = $1.LTE($3)        
        if err != NoError {
            return err
        }
    }
    | expr GTE expr
    { 
        var err int
        $$, err = $1.GTE($3)                
        if err != NoError {
            return err
        }
    }
    | expr MATCH expr
    {
        var err int
        $$, err = $1.MATCH($3)
        if err != NoError {
            return err
        }
    }
    | expr '+' expr
    { 
        var err int
        $$, err = $1.Add($3) 
        if err != NoError {
            return err
        }
    }
	| expr '-' expr
    { 
        var err int
        $$, err = $1.Sub($3)
        if err != NoError {
            return err
        }
    }
	| expr '*' expr
    {
        var err int
        $$, err = $1.Multi($3)
        if err != NoError {
            return err
        }
    }
	| expr '/' expr
    {
        var err int
        $$, err = $1.Div($3)
        if err != NoError {
            return err
        }
    }
	| expr '%' expr
    {
        var err int
        $$, err = $1.Mod($3)
        if err != NoError {
            return err
        }
    }
    | '-' expr
    {
        var err int
        $$, err = NewValue(0).Sub($2)
        if err != NoError {
            return err
        }
    }
    | VALUE
    { 
        $$ = $1 
    }
    ;
%%