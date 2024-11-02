package game

/// ======== No effect natures ========

var NATURE_HARDY = Nature{
	"Hardy",
	[5]float32{1, 1, 1, 1, 1},
}

var NATURE_DOCILE = Nature{
	"Docile",
	[5]float32{1, 1, 1, 1, 1},
}

var NATURE_BASHFUL = Nature{
	"Bashful",
	[5]float32{1, 1, 1, 1, 1},
}

var NATURE_QUIRKY = Nature{
	"Quirky",
	[5]float32{1, 1, 1, 1, 1},
}

var NATURE_SERIOUS = Nature{
	"Serious",
	[5]float32{1, 1, 1, 1, 1},
}

/// ======== -Attack Natures ========

var NATURE_BOLD = Nature{
	"Bold",
	[5]float32{.9, 1.1, 1, 1, 1},
}

var NATURE_MODEST = Nature{
	"Modest",
	[5]float32{.9, 1, 1.1, 1, 1},
}

var NATURE_CALM = Nature{
	"Calm",
	[5]float32{.9, 1, 1, 1.1, 1},
}

var NATURE_TIMID = Nature{
	"Timid",
	[5]float32{.9, 1, 1, 1, 1.1},
}

/// ======== -Defense Natures ========

var NATURE_LONELY = Nature{
	"Lonely",
	[5]float32{1.1, .9, 1, 1, 1},
}

var NATURE_MILD = Nature{
	"Mild",
	[5]float32{1, .9, 1.1, 1, 1},
}

var NATURE_GENTLE = Nature{
	"Gentle",
	[5]float32{1, .9, 1, 1.1, 1},
}

var NATURE_HASTY = Nature{
	"Hasty",
	[5]float32{1, .9, 1, 1, 1.1},
}

/// ======== -SpAttack Natures ========

var NATURE_ADAMENT = Nature{
	"Adament",
	[5]float32{1.1, 1, .9, 1, 1},
}

var NATURE_IMPISH = Nature{
	"Impish",
	[5]float32{1, 1.1, .9, 1, 1},
}

var NATURE_CAREFUL = Nature{
	"Careful",
	[5]float32{1, 1, .9, 1.1, 1},
}

var NATURE_JOLLY = Nature{
	"Jolly",
	[5]float32{1, 1, .9, 1, 1.1},
}

/// ======== -SpDef Natures ========

var NATURE_NAUGHTY = Nature{
	"Naughty",
	[5]float32{1.1, 1, 1, .9, 1},
}

var NATURE_LAX = Nature{
	"LAX",
	[5]float32{1, 1.1, 1, .9, 1},
}

var NATURE_RASH = Nature{
	"Rash",
	[5]float32{1, 1, 1.1, .9, 1},
}

var NATURE_NAIVE = Nature{
	"Naive",
	[5]float32{1, 1, 1, .9, 1.1},
}

/// ======== -SpDef Natures ========

var NATURE_BRAVE = Nature{
	"Brave",
	[5]float32{1.1, 1, 1, 1, .9},
}

var NATURE_RELAXED = Nature{
	"Relaxed",
	[5]float32{1, 1.1, 1, 1, .9},
}

var NATURE_QUIET = Nature{
	"Quiet",
	[5]float32{1, 1, 1.1, 1, .9},
}

var NATURE_SASSY = Nature{
	"Sassy",
	[5]float32{1, 1, 1, 1.1, .9},
}

var NATURES = [...]Nature{
	NATURE_HARDY,
	NATURE_DOCILE,
	NATURE_BASHFUL,
	NATURE_QUIRKY,
	NATURE_SERIOUS,
	NATURE_BOLD,
	NATURE_MODEST,
	NATURE_CALM,
	NATURE_TIMID,
	NATURE_LONELY,
	NATURE_MILD,
	NATURE_GENTLE,
	NATURE_HASTY,
	NATURE_ADAMENT,
	NATURE_IMPISH,
	NATURE_CAREFUL,
	NATURE_JOLLY,
	NATURE_NAUGHTY,
	NATURE_LAX,
	NATURE_RASH,
	NATURE_NAIVE,
	NATURE_BRAVE,
	NATURE_RELAXED,
	NATURE_QUIET,
	NATURE_SASSY,
}

var TYPE_MAP = map[string]*PokemonType{
	TYPENAME_NORMAL:   &TYPE_NORMAL,
	TYPENAME_FIRE:     &TYPE_FIRE,
	TYPENAME_WATER:    &TYPE_WATER,
	TYPENAME_ELECTRIC: &TYPE_ELECTRIC,
	TYPENAME_GRASS:    &TYPE_GRASS,
	TYPENAME_ICE:      &TYPE_ICE,
	TYPENAME_FIGHTING: &TYPE_FIGHTING,
	TYPENAME_POISON:   &TYPE_POISON,
	TYPENAME_GROUND:   &TYPE_GROUND,
	TYPENAME_FLYING:   &TYPE_FLYING,
	TYPENAME_PSYCHIC:  &TYPE_PSYCHIC,
	TYPENAME_BUG:      &TYPE_BUG,
	TYPENAME_ROCK:     &TYPE_ROCK,
	TYPENAME_GHOST:    &TYPE_GHOST,
	TYPENAME_DRAGON:   &TYPE_DRAGON,
	TYPENAME_DARK:     &TYPE_DARK,
	TYPENAME_STEEL:    &TYPE_STEEL,
	TYPENAME_FAIRY:    &TYPE_FAIRY,
}

var TYPE_NORMAL = PokemonType{
	TYPENAME_NORMAL,
	map[string]float32{
		TYPENAME_ROCK:  0.5,
		TYPENAME_STEEL: 0.5,

		TYPENAME_GHOST: 0,
	},
}

var TYPE_FIRE = PokemonType{
	TYPENAME_FIRE,
	map[string]float32{
		TYPENAME_GRASS: 2,
		TYPENAME_ICE:   2,
		TYPENAME_BUG:   2,
		TYPENAME_STEEL: 2,

		TYPENAME_FIRE:   .5,
		TYPENAME_WATER:  .5,
		TYPENAME_ROCK:   .5,
		TYPENAME_DRAGON: .5,
	},
}

var TYPE_WATER = PokemonType{
	TYPENAME_WATER,
	map[string]float32{
		TYPENAME_FIRE:   2,
		TYPENAME_GROUND: 2,
		TYPENAME_ROCK:   2,

		TYPENAME_WATER:  .5,
		TYPENAME_GRASS:  .5,
		TYPENAME_DRAGON: .5,
	},
}

var TYPE_ELECTRIC = PokemonType{
	TYPENAME_ELECTRIC,
	map[string]float32{
		TYPENAME_WATER:  2,
		TYPENAME_FLYING: 2,

		TYPENAME_ELECTRIC: .5,
		TYPENAME_GRASS:    .5,
		TYPENAME_DRAGON:   .5,

		TYPENAME_GROUND: 0,
	},
}

var TYPE_GRASS = PokemonType{
	TYPENAME_GRASS,
	map[string]float32{
		TYPENAME_WATER:  2,
		TYPENAME_GROUND: 2,
		TYPENAME_ROCK:   2,

		TYPENAME_FIRE:   .5,
		TYPENAME_GRASS:  .5,
		TYPENAME_POISON: .5,
		TYPENAME_FLYING: .5,
		TYPENAME_BUG:    .5,
		TYPENAME_DRAGON: .5,
		TYPENAME_STEEL:  .5,
	},
}

var TYPE_ICE = PokemonType{
	TYPENAME_ICE,
	map[string]float32{
		TYPENAME_GRASS:  2,
		TYPENAME_GROUND: 2,
		TYPENAME_FLYING: 2,
		TYPENAME_DRAGON: 2,

		TYPENAME_FIRE:  .5,
		TYPENAME_WATER: .5,
		TYPENAME_ICE:   .5,
		TYPENAME_STEEL: .5,
	},
}

var TYPE_FIGHTING = PokemonType{
	TYPENAME_FIGHTING,
	map[string]float32{
		TYPENAME_NORMAL: 2,
		TYPENAME_ICE:    2,
		TYPENAME_ROCK:   2,
		TYPENAME_DARK:   2,
		TYPENAME_STEEL:  2,

		TYPENAME_POISON:  .5,
		TYPENAME_FLYING:  .5,
		TYPENAME_PSYCHIC: .5,
		TYPENAME_BUG:     .5,
		TYPENAME_FAIRY:   .5,

		TYPENAME_GHOST: 0,
	},
}

var TYPE_POISON = PokemonType{
	TYPENAME_POISON,
	map[string]float32{
		TYPENAME_GRASS: 2,
		TYPENAME_FAIRY: 2,

		TYPENAME_POISON: .5,
		TYPENAME_GROUND: .5,
		TYPENAME_ROCK:   .5,
		TYPENAME_GHOST:  .5,

		TYPENAME_STEEL: 0,
	},
}

var TYPE_GROUND = PokemonType{
	TYPENAME_GROUND,
	map[string]float32{
		TYPENAME_FIRE:     2,
		TYPENAME_ELECTRIC: 2,
		TYPENAME_POISON:   2,
		TYPENAME_ROCK:     2,
		TYPENAME_STEEL:    2,

		TYPENAME_GRASS: .5,
		TYPENAME_BUG:   .5,

		TYPENAME_FLYING: 0,
	},
}

var TYPE_FLYING = PokemonType{
	TYPENAME_FLYING,
	map[string]float32{
		TYPENAME_GRASS:    2,
		TYPENAME_FIGHTING: 2,
		TYPENAME_BUG:      2,

		TYPENAME_ELECTRIC: .5,
		TYPENAME_ROCK:     .5,
		TYPENAME_STEEL:    .5,
	},
}

var TYPE_PSYCHIC = PokemonType{
	TYPENAME_PSYCHIC,
	map[string]float32{
		TYPENAME_FIGHTING: 2,
		TYPENAME_POISON:   2,

		TYPENAME_PSYCHIC: .5,
		TYPENAME_STEEL:   .5,

		TYPENAME_DARK: .5,
	},
}

var TYPE_BUG = PokemonType{
	TYPENAME_BUG,
	map[string]float32{
		TYPENAME_GRASS:   2,
		TYPENAME_PSYCHIC: 2,
		TYPENAME_DARK:    2,

		TYPENAME_FIRE:     .5,
		TYPENAME_FIGHTING: .5,
		TYPENAME_POISON:   .5,
		TYPENAME_FLYING:   .5,
		TYPENAME_GHOST:    .5,
		TYPENAME_STEEL:    .5,
		TYPENAME_FAIRY:    .5,
	},
}

var TYPE_ROCK = PokemonType{
	TYPENAME_ROCK,
	map[string]float32{
		TYPENAME_FIRE:   2,
		TYPENAME_ICE:    2,
		TYPENAME_FLYING: 2,
		TYPENAME_BUG:    2,

		TYPENAME_FIGHTING: .5,
		TYPENAME_GROUND:   .5,
		TYPENAME_STEEL:    .5,
	},
}

var TYPE_GHOST = PokemonType{
	TYPENAME_GHOST,
	map[string]float32{
		TYPENAME_PSYCHIC: 2,
		TYPENAME_GHOST:   2,

		TYPENAME_DARK: .5,

		TYPENAME_NORMAL: 0,
	},
}

var TYPE_DRAGON = PokemonType{
	TYPENAME_DRAGON,
	map[string]float32{
		TYPENAME_DRAGON: 2,

		TYPENAME_STEEL: .5,

		TYPENAME_FAIRY: 0,
	},
}

var TYPE_DARK = PokemonType{
	TYPENAME_DARK,
	map[string]float32{
		TYPENAME_PSYCHIC: 2,
		TYPENAME_GHOST:   2,

		TYPENAME_FIGHTING: .5,
		TYPENAME_DARK:     .5,
		TYPENAME_FAIRY:    .5,
	},
}

var TYPE_STEEL = PokemonType{
	TYPENAME_STEEL,
	map[string]float32{
		TYPENAME_ICE:   2,
		TYPENAME_ROCK:  2,
		TYPENAME_FAIRY: 2,

		TYPENAME_FIRE:     .5,
		TYPENAME_WATER:    .5,
		TYPENAME_ELECTRIC: .5,
		TYPENAME_STEEL:    .5,
	},
}

var TYPE_FAIRY = PokemonType{
	TYPENAME_FAIRY,
	map[string]float32{
		TYPENAME_FIGHTING: 2,
		TYPENAME_DRAGON:   2,
		TYPENAME_DARK:     2,

		TYPENAME_FIRE:   .5,
		TYPENAME_POISON: .5,
		TYPENAME_STEEL:  .5,
	},
}
