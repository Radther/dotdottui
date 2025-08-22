package tui

// InitializeMockTasks returns a set of sample tasks with a coherent real-world theme
// following hierarchical nesting, varied task statuses, and proper tagging for UI testing.
// The theme centers around baking a cake, providing a relatable and structured workflow.
func InitializeMockTasks() []Task {
	return []Task{
		NewTask("Gather all ingredients from pantry", Done),
		NewTask("Prepare cake batter", Active,
			NewTask("Sift flour and baking powder together", Done),
			NewTask("Cream butter and sugar until fluffy", Active),
			NewTask("Add eggs one at a time", Todo),
			NewTask("Alternate flour mixture and milk", Todo),
		),
		NewTask("Preheat oven to 350°F (175°C)", Todo),
		NewTask("Grease and flour two 9-inch round cake pans", Active),
		NewTask("Add vanilla extract and mix gently to avoid overmixing the delicate cake batter #flavor", Todo),
		NewTask("Prepare workspace and tools #setup", Done),
		NewTask("Set timer for baking duration #timing", Todo),
		NewTask("Bake and finish cake #baking", Todo,
			NewTask("Bake cake layers", Todo,
				NewTask("Pour batter evenly into prepared pans", Todo),
				NewTask("Bake for 25-30 minutes until golden", Todo),
				NewTask("Test doneness with toothpick", Todo),
			),
			NewTask("Cool cakes completely on wire racks", Todo),
			NewTask("Make buttercream frosting", Todo),
			NewTask("Assemble and decorate cake", Todo),
		),
		NewTask("Clean up kitchen and wash dishes #cleanup", Todo),
		NewTask("Take photos of finished cake #memories", Todo),
	}
}

// GetAlternativeMockTasks returns a different set of mock tasks with a gardening theme
// for variety in testing scenarios.
func GetAlternativeMockTasks() []Task {
	return []Task{
		NewTask("Plan spring garden layout #planning", Done),
		NewTask("Prepare garden beds", Active,
			NewTask("Clear winter debris and weeds", Done),
			NewTask("Test soil pH and nutrients", Active),
			NewTask("Add compost and fertilizer", Todo),
			NewTask("Till the soil to proper depth", Todo),
		),
		NewTask("Start seeds indoors #seedlings", Todo,
			NewTask("Prepare seed starting setup", Todo,
				NewTask("Set up grow lights", Todo),
				NewTask("Prepare seed trays with soil", Todo),
				NewTask("Label varieties and planting dates", Todo),
			),
			NewTask("Plant tomato seeds", Todo),
			NewTask("Plant pepper seeds", Todo),
			NewTask("Plant herb seeds", Todo),
		),
		NewTask("Order garden supplies #shopping", Active),
		NewTask("Install irrigation system #watering", Todo),
		NewTask("Monitor and care for seedlings #maintenance", Todo,
			NewTask("Check moisture levels daily", Todo),
			NewTask("Thin overcrowded seedlings", Todo),
			NewTask("Transplant to larger containers when ready", Todo),
		),
	}
}

// GetMinimalMockTasks returns a simple set of tasks for basic testing
func GetMinimalMockTasks() []Task {
	return []Task{
		NewTask("First task", Done),
		NewTask("Second task", Active),
		NewTask("Third task", Todo),
		NewTask("Fourth task with subtasks", Todo,
			NewTask("Subtask 1", Todo),
			NewTask("Subtask 2", Active),
		),
	}
}